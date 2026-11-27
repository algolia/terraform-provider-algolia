"""Check the tagged commit's Tests run before signing a release."""

import json
import os
import subprocess
import time


# These are the required job names from .github/workflows/test.yml.
REQUIRED_JOBS = ("Build", "Lint", "Unit Tests", "Acceptance Tests", "E2E Tests")


def gh(*args):
    try:
        return json.loads(subprocess.check_output(["gh", *args]))
    except subprocess.CalledProcessError as error:
        raise SystemExit(f"GitHub CLI failed with exit code {error.returncode}. See its error above.") from None
    except json.JSONDecodeError:
        raise SystemExit("GitHub CLI returned invalid JSON.") from None


def verify_jobs(run):
    path = f"repos/{os.environ['GH_REPO']}/actions/runs/{run['databaseId']}"
    remaining = set(REQUIRED_JOBS)
    # A partial rerun keeps successful jobs from earlier attempts.
    for attempt in range(run["attempt"], 0, -1):
        pages = gh("api", "--paginate", "--slurp", f"{path}/attempts/{attempt}/jobs")
        jobs = [job for page in pages for job in page["jobs"]]
        for name in sorted(remaining):
            results = [job["conclusion"] for job in jobs if job["name"] == name]
            if not results:
                continue
            if results != ["success"]:
                raise SystemExit(f"Expected one successful {name} job, got {results}: {run['url']}")
            remaining.remove(name)
        if not remaining:
            return
    raise SystemExit(f"Missing required jobs: {', '.join(sorted(remaining))}: {run['url']}")


def main():
    sha = os.environ["SHA"]
    # A tag pushed just after a merge may arrive before Tests finishes.
    deadline = time.monotonic() + 25 * 60
    while True:
        runs = gh(
            "run", "list", "--workflow=test.yml", "--commit", sha,
            "--branch", "main", "--event", "push", "--limit", "1",
            "--json", "status,conclusion,databaseId,attempt,url",
        )
        status = runs[0]["status"] if runs else "not listed yet"
        if status == "completed":
            run = runs[0]
            if run["conclusion"] != "success":
                raise SystemExit(f"Tests for {sha} concluded '{run['conclusion']}': {run['url']}")
            verify_jobs(run)
            print(f"All required Tests jobs passed for {sha}.")
            return
        if time.monotonic() >= deadline:
            if not runs:
                raise SystemExit(f"No Tests run after 25 minutes. Was {sha} pushed to main?")
            raise SystemExit(f"Tests still {status} after 25 minutes. Retry after {runs[0]['url']} finishes.")
        print(f"Tests for {sha}: {status}. Waiting 30 seconds.", flush=True)
        time.sleep(30)


if __name__ == "__main__":
    try:
        main()
    except SystemExit as error:
        print(f"::error::{error}")
        raise SystemExit(1) from None
