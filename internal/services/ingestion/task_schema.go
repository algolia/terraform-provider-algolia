package ingestion

import (
	"strings"

	ingestionapi "github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func taskResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Algolia Ingestion task resource: a scheduled or on-demand pipeline that reads " +
			"records from a source, optionally transforms them, and writes them to a destination. The Ingestion " +
			"API is region-routed, so the provider's `analytics_region` (or the `ALGOLIA_ANALYTICS_REGION` " +
			"environment variable) must be configured.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `task_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"task_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the `algolia_ingestion_source` this task " +
					"reads from. Changing this forces replacement: the Ingestion API's task update endpoint has " +
					"no way to change a task's source after creation.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination_id": schema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the `algolia_ingestion_destination` this " +
					"task writes to.",
				Required: true,
			},
			"action": schema.StringAttribute{
				Description: "Action to perform on the destination index for each record. One of: " +
					strings.Join(allowedActionTypeStrings(), ", ") + ". Changing this forces replacement: the " +
					"Ingestion API's task update endpoint has no way to change a task's action after creation. " +
					"Note the API does not return this field when reading a task back, so it cannot be " +
					"recovered by `terraform import`: set it in configuration after importing, which will " +
					"plan a replacement.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedActionTypeStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subscription_action": schema.StringAttribute{
				Description: "Action to perform on the destination index for records ingested through a " +
					"streaming/subscription-based source. One of: " + strings.Join(allowedActionTypeStrings(), ", ") + ". " +
					"Changing the value updates the task in place, but removing the attribute forces " +
					"replacement: the Ingestion API's task update endpoint can set this field and change " +
					"it, but has no way to clear it.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf(allowedActionTypeStrings()...),
				},
				PlanModifiers: []planmodifier.String{
					requiresReplaceOnRemoval(),
				},
			},
			"cron": schema.StringAttribute{
				Description: "Cron expression for the task's schedule (e.g. `0 0 * * *` for daily). Omit for an " +
					"on-demand task that only runs when triggered manually.\n\n" +
					"Changing the schedule updates the task in place. Removing `cron` altogether forces " +
					"replacement, because the Ingestion API has no way to clear a schedule: an empty " +
					"expression is rejected as invalid and a null one is ignored, so a task can only " +
					"become on-demand by being recreated. To stop a scheduled task from running without " +
					"recreating it, set `enabled = false` instead - that keeps the schedule and is almost " +
					"always what is wanted.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					requiresReplaceOnRemoval(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the task is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"failure_threshold": schema.Int64Attribute{
				Description: "Maximum accepted percentage of failures for a task run to finish successfully. " +
					"Computed because the API substitutes its own default when this is omitted.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"input": schema.StringAttribute{
				Description: "JSON-encoded configuration for the task's input, when its source's type needs one " +
					"(e.g. `jsonencode({ streams = [...] })` for a Docker-based source). Not every task needs " +
					"input - a task on a \"push\" source, for example, has none, so `input` may be omitted. The " +
					"Ingestion API returns a task's `input` in full when reading it back (nothing is redacted), " +
					"so this attribute is refreshed on read. To avoid a perpetual diff caused by harmless JSON " +
					"differences (key order, array order), the refresh only replaces the configured value when it " +
					"is not semantically equivalent to what the API returned.",
				Optional: true,
			},
			"notifications": schema.StringAttribute{
				Description: "JSON-encoded notification settings, e.g. `jsonencode({ email = { enabled = true } " +
					"})`. Refreshed on read using the same semantic-equality preservation as `input`. Computed " +
					"because the API substitutes its own defaults when this is omitted.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policies": schema.StringAttribute{
				Description: "JSON-encoded task policies, e.g. `jsonencode({ criticalThreshold = 5 })`. The API " +
					"caps `criticalThreshold` at 10 and rejects anything higher. " +
					"Refreshed on read using the same semantic-equality preservation as `input`. Computed " +
					"because the API substitutes its own defaults when this is omitted.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cursor": schema.StringAttribute{
				Description: "Date and time when the last cursor was created, in RFC 3339 format; used to resume " +
					"a streaming task from a specific point. The Ingestion API's task update endpoint has no way " +
					"to change this after creation, and its true value advances automatically as the task runs " +
					"(a runtime concern outside this provider's scope) - so, unlike " +
					"`input`/`notifications`/`policies`, it is never refreshed from the API on read; the " +
					"configured value (or null, if omitted) is always preserved as-is. Because it can only be set " +
					"at creation, changing it forces the task to be replaced. Not recoverable on import.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time when the resource was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time when the resource was last updated, in RFC 3339 format.",
				Computed:    true,
			},
			"last_run": schema.StringAttribute{
				Description: "Date and time of the last time the task ran, in RFC 3339 format. Null if the task " +
					"has never run.",
				Computed: true,
			},
			"next_run": schema.StringAttribute{
				Description: "Date and time of the next scheduled run of the task, in RFC 3339 format. Null for " +
					"on-demand tasks or tasks without a `cron` schedule.",
				Computed: true,
			},
		},
	}
}

// allowedActionTypeStrings derives the list of valid `action`/
// `subscription_action` values from the Go client's ActionType enum
// rather than hard-coding it, so a new action type added upstream doesn't
// require a provider code change to become selectable (only a client
// bump).
func allowedActionTypeStrings() []string {
	values := make([]string, 0, len(ingestionapi.AllowedActionTypeEnumValues))
	for _, v := range ingestionapi.AllowedActionTypeEnumValues {
		values = append(values, string(v))
	}

	return values
}
