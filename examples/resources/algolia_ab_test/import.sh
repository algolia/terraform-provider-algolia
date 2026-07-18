# Import an A/B test by its numeric ID. Metrics are not recoverable and
# several attributes force a replace on the next plan; see the resource docs.
terraform import algolia_ab_test.ranking_experiment 42
