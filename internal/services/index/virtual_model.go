package index

import "github.com/hashicorp/terraform-plugin-framework/types"

type VirtualIndexResourceModel struct {
	Name               types.String `tfsdk:"name"`
	PrimaryIndexName   types.String `tfsdk:"primary_index_name"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`
	Entries            types.Int64  `tfsdk:"entries"`
	DataSize           types.Int64  `tfsdk:"data_size"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	Attributes         types.Object `tfsdk:"attributes"`
	Ranking            types.Object `tfsdk:"ranking"`
	Faceting           types.Object `tfsdk:"faceting"`
	Highlighting       types.Object `tfsdk:"highlighting"`
	Pagination         types.Object `tfsdk:"pagination"`
	Typos              types.Object `tfsdk:"typos"`
	Languages          types.Object `tfsdk:"languages"`
	QueryStrategy      types.Object `tfsdk:"query_strategy"`
	Performance        types.Object `tfsdk:"performance"`
	Advanced           types.Object `tfsdk:"advanced"`
}

type VirtualIndexDataSourceModel struct {
	Name             types.String `tfsdk:"name"`
	PrimaryIndexName types.String `tfsdk:"primary_index_name"`
	Entries          types.Int64  `tfsdk:"entries"`
	DataSize         types.Int64  `tfsdk:"data_size"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	Attributes       types.Object `tfsdk:"attributes"`
	Ranking          types.Object `tfsdk:"ranking"`
	Faceting         types.Object `tfsdk:"faceting"`
	Highlighting     types.Object `tfsdk:"highlighting"`
	Pagination       types.Object `tfsdk:"pagination"`
	Typos            types.Object `tfsdk:"typos"`
	Languages        types.Object `tfsdk:"languages"`
	QueryStrategy    types.Object `tfsdk:"query_strategy"`
	Performance      types.Object `tfsdk:"performance"`
	Advanced         types.Object `tfsdk:"advanced"`
}

func virtualToIndexModel(model VirtualIndexResourceModel) IndexResourceModel {
	return IndexResourceModel{
		Name:               model.Name,
		DeletionProtection: model.DeletionProtection,
		Primary:            model.PrimaryIndexName,
		Entries:            model.Entries,
		DataSize:           model.DataSize,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
		Attributes:         model.Attributes,
		Ranking:            model.Ranking,
		Faceting:           model.Faceting,
		Highlighting:       model.Highlighting,
		Pagination:         model.Pagination,
		Typos:              model.Typos,
		Languages:          model.Languages,
		QueryStrategy:      model.QueryStrategy,
		Performance:        model.Performance,
		Advanced:           model.Advanced,
	}
}

func virtualFromIndexModel(indexModel IndexResourceModel, model *VirtualIndexResourceModel) {
	model.Name = indexModel.Name
	model.PrimaryIndexName = indexModel.Primary
	model.DeletionProtection = indexModel.DeletionProtection
	model.Entries = indexModel.Entries
	model.DataSize = indexModel.DataSize
	model.CreatedAt = indexModel.CreatedAt
	model.UpdatedAt = indexModel.UpdatedAt
	model.Attributes = indexModel.Attributes
	model.Ranking = indexModel.Ranking
	model.Faceting = indexModel.Faceting
	model.Highlighting = indexModel.Highlighting
	model.Pagination = indexModel.Pagination
	model.Typos = indexModel.Typos
	model.Languages = indexModel.Languages
	model.QueryStrategy = indexModel.QueryStrategy
	model.Performance = indexModel.Performance
	model.Advanced = indexModel.Advanced
}

func virtualDataSourceFromIndexModel(indexModel IndexResourceModel, model *VirtualIndexDataSourceModel) {
	model.Name = indexModel.Name
	model.PrimaryIndexName = indexModel.Primary
	model.Entries = indexModel.Entries
	model.DataSize = indexModel.DataSize
	model.CreatedAt = indexModel.CreatedAt
	model.UpdatedAt = indexModel.UpdatedAt
	model.Attributes = indexModel.Attributes
	model.Ranking = indexModel.Ranking
	model.Faceting = indexModel.Faceting
	model.Highlighting = indexModel.Highlighting
	model.Pagination = indexModel.Pagination
	model.Typos = indexModel.Typos
	model.Languages = indexModel.Languages
	model.QueryStrategy = indexModel.QueryStrategy
	model.Performance = indexModel.Performance
	model.Advanced = indexModel.Advanced
}

