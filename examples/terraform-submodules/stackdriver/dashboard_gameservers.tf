
resource "google_monitoring_dashboard" "gameservers" {
  dashboard_json = <<EOF
{
  "displayName": "GameServers",
      "gridLayout": {
        "columns": "2",
        "widgets": [
          {
            "title": "agones/gameservers_count",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_MEAN",
                        "groupByFields": [
                          "metric.label.\"type\""
                        ]
                      },
                      "secondaryAggregation": {}
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/fleets_replicas_count [SUM]",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/fleets_replicas_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MAX"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/gameservers_count",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_SUM"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/gameservers_node_count_bucket per node",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_node_count_bucket\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_MEAN",
                        "groupByFields": [
                          "metric.label.\"type\""
                        ]
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                },
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/nodes_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_MEAN"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/k8s_client_http_request_total",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/k8s_client_http_request_total\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_RATE"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "GS Allocation - gameserver_allocations_duration_seconds",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameserver_allocations_duration_seconds\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_DELTA",
                        "crossSeriesReducer": "REDUCE_PERCENTILE_50"
                      }
                    },
                    "unitOverride": "s"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/fleet_allocations_count",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/fleet_allocations_count\" metric.label.\"fleet_name\"=\"\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "games",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameserver_allocations_duration_seconds\" resource.type=\"k8s_container\" metric.label.\"fleet_name\"=\"simple-udp\" metric.label.\"status\"=\"UnAllocated\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_DELTA",
                        "crossSeriesReducer": "REDUCE_PERCENTILE_50"
                      }
                    },
                    "unitOverride": "s"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/gameserver_ready_duration [STD. DEV.]",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameserver_ready_duration\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_DELTA",
                        "crossSeriesReducer": "REDUCE_STDDEV"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones GS Ready duration",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameserver_ready_duration\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_DELTA",
                        "crossSeriesReducer": "REDUCE_COUNT"
                      }
                    },
                    "unitOverride": "ms"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "NewOne",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/gameservers_count by label.fleet_name [SUM]",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_SUM",
                        "groupByFields": [
                          "metric.label.\"fleet_name\""
                        ]
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "GS",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_SUM"
                      }
                    },
                    "unitOverride": "1"
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          },
          {
            "title": "agones/gameservers_count (filtered) (grouped)",
            "xyChart": {
              "dataSets": [
                {
                  "timeSeriesQuery": {
                    "timeSeriesFilter": {
                      "filter": "metric.type=\"custom.googleapis.com/opencensus/gameservers_count\" resource.type=\"k8s_container\" metric.label.\"fleet_name\"=\"fleet-abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv-3d386978\"",
                      "aggregation": {
                        "perSeriesAligner": "ALIGN_MEAN",
                        "crossSeriesReducer": "REDUCE_MEAN",
                        "groupByFields": [
                          "metric.label.\"type\""
                        ]
                      },
                      "secondaryAggregation": {}
                    }
                  },
                  "plotType": "LINE",
                  "minAlignmentPeriod": "60s"
                }
              ],
              "timeshiftDuration": "0s",
              "yAxis": {
                "label": "y1Axis",
                "scale": "LINEAR"
              },
              "chartOptions": {
                "mode": "COLOR"
              }
            }
          }
        ]
      }
}

EOF
}