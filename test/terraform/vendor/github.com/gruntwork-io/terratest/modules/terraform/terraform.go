// Package terraform allows to interact with Terraform.
package terraform

// https://www.terraform.io/docs/commands/plan.html#detailed-exitcode

// TerraformPlanChangesPresentExitCode is the exit code returned by terraform plan detailed exitcode when changes are present
const TerraformPlanChangesPresentExitCode = 2

// DefaultSuccessExitCode is the exit code returned when terraform command succeeds
const DefaultSuccessExitCode = 0

// DefaultErrorExitCode is the exit code returned when terraform command fails
const DefaultErrorExitCode = 1
