locals {
  app_name = "passenger-datadog-monitor"
}

module "ecr" {
  source  = "ibdolphin.jfrog.io/terraform__ibotta/ecr/aws"
  version = "4.0.0"

  service_name         = local.app_name
  git_tagged_lifecycle = true
  tags                 = module.tags.tags
}