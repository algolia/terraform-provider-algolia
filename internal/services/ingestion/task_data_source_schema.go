package ingestion

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func taskDataSourceSchema() datasourceschema.Schema {
	return datasourceschema.Schema{
		Description: "Use this data source to read an Algolia Ingestion task's configuration, including its " +
			"`input`, `notifications`, `policies`, and `cursor` in full: the Ingestion API does not redact a " +
			"task's configuration.",
		Attributes: map[string]datasourceschema.Attribute{
			"task_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the task to read.",
				Required:    true,
			},
			"id": datasourceschema.StringAttribute{
				Description: "Terraform identifier for the resource. Equal to `task_id`.",
				Computed:    true,
			},
			"source_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the source this task reads from.",
				Computed:    true,
			},
			"destination_id": datasourceschema.StringAttribute{
				Description: "Universally unique identifier (UUID) of the destination this task writes to.",
				Computed:    true,
			},
			"action": datasourceschema.StringAttribute{
				Description: "Action to perform on the destination index for each record.",
				Computed:    true,
			},
			"subscription_action": datasourceschema.StringAttribute{
				Description: "Action to perform on the destination index for records ingested through a " +
					"streaming/subscription-based source.",
				Computed: true,
			},
			"cron": datasourceschema.StringAttribute{
				Description: "Cron expression for the task's schedule. Null for on-demand tasks.",
				Computed:    true,
			},
			"enabled": datasourceschema.BoolAttribute{
				Description: "Whether the task is enabled.",
				Computed:    true,
			},
			"failure_threshold": datasourceschema.Int64Attribute{
				Description: "Maximum accepted percentage of failures for a task run to finish successfully.",
				Computed:    true,
			},
			"input": datasourceschema.StringAttribute{
				Description: "JSON-encoded configuration for the task's input. Null if the task needs no input.",
				Computed:    true,
			},
			"notifications": datasourceschema.StringAttribute{
				Description: "JSON-encoded notification settings.",
				Computed:    true,
			},
			"policies": datasourceschema.StringAttribute{
				Description: "JSON-encoded task policies.",
				Computed:    true,
			},
			"cursor": datasourceschema.StringAttribute{
				Description: "Date and time when the last cursor was created, in RFC 3339 format.",
				Computed:    true,
			},
			"created_at": datasourceschema.StringAttribute{
				Description: "Date and time when the resource was created, in RFC 3339 format.",
				Computed:    true,
			},
			"updated_at": datasourceschema.StringAttribute{
				Description: "Date and time when the resource was last updated, in RFC 3339 format.",
				Computed:    true,
			},
			"last_run": datasourceschema.StringAttribute{
				Description: "Date and time of the last time the task ran, in RFC 3339 format.",
				Computed:    true,
			},
			"next_run": datasourceschema.StringAttribute{
				Description: "Date and time of the next scheduled run of the task, in RFC 3339 format.",
				Computed:    true,
			},
		},
	}
}
