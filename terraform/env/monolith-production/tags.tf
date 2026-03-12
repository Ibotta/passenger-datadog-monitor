module "tags" {
  source  = "ibdolphin.jfrog.io/terraform__ibotta/tags/any"
  version = "1.1.1"

  account_alias = "monolith-production"
  service      = local.app_name
  service_tier = "2"
  data_classification = "2"
  pi = "no"

  domain       = "proof-of-concept"
  squad        = "shared"
  subgroup     = "shared"
  group        = "shared"

  custom_tags = {
    application = local.app_name
  }
}