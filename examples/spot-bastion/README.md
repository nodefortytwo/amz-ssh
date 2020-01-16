#Simple Bastion Server

## Providers

| Name | Version |
|------|---------|
| aws | n/a |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:-----:|
| instance\_type | Instance type to use for the bastion server | `string` | `"t3a.nano"` | no |
| subnet\_id | (public} subnet to launch the instance in | `string` | n/a | yes |
| tags | tags to add to resources created | `map(string)` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| sg\_id | n/a |

