// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

// This file is intentionally empty but kept for future test implementations.
// When adding acceptance tests for resources and data sources, use the following pattern:
//
// import (
//     "github.com/hashicorp/terraform-plugin-framework/providerserver"
//     "github.com/hashicorp/terraform-plugin-go/tfprotov6"
// )
//
// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
//     "strato": providerserver.NewProtocol6WithError(New("test")()),
// }
