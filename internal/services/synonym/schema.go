package synonym

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func synonymResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages a single Algolia synonym object in an index.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Terraform identifier in the form <index_name>/<object_id>.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"index_name": schema.StringAttribute{
				Description: "The index that owns the synonym.",
				Required:    true,
			},
			"object_id": schema.StringAttribute{
				Description: "Unique identifier of the synonym object.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Synonym type.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("synonym", "oneWaySynonym", "altCorrection1", "altCorrection2", "placeholder"),
				},
			},
			"synonyms": schema.SetAttribute{
				Description: "Synonyms used by regular and one-way synonyms.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"input": schema.StringAttribute{
				Description: "Base input used by one-way synonyms.",
				Optional:    true,
			},
			"word": schema.StringAttribute{
				Description: "Base word used by alternative correction synonyms.",
				Optional:    true,
			},
			"corrections": schema.SetAttribute{
				Description: "Corrections used by alternative correction synonyms.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"placeholder": schema.StringAttribute{
				Description: "Placeholder token used by placeholder synonyms.",
				Optional:    true,
			},
			"replacements": schema.SetAttribute{
				Description: "Replacement values used by placeholder synonyms.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

