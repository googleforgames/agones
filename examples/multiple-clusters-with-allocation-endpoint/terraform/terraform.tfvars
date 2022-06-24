project        = "your-example-project"
name           = "agones-cluster"
agones_version = ""
machine_type   = "e2-standard-4"

// Note: This is the number of gameserver nodes. The Agones module will automatically create an additional
// two node pools with 1 node each for "agones-system" and "agones-metrics".
node_count               = "2"
network                  = "agones-demo"
subnetwork               = ""
log_level                = "info"
feature_gates            = "" //https://agones.dev/site/docs/guides/feature-stages/#feature-gates
windows_node_count       = "0"
windows_machine_type     = "e2-standard-4"
values_file              = ""
enableAllocationEndpoint = true
service_account_name     = "agones-gameservers-node-pool"
min_node_count           = 1
max_node_count           = 5
# add more regions, zones, cidr_ranges if you want to use more than two regions
region_1                 = "europe-west1"
region_2                 = "europe-central2"
zone_1                   = "europe-west1-c"
zone_2                   = "europe-central2-b"
cidr_range_1             = "10.140.0.0/20"
cidr_range_2             = "10.146.0.0/20"