// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use crate::errors::Result;
use tonic::transport::Channel;

mod api {
    tonic::include_proto!("agones.dev.sdk.beta");
}

use api::sdk_client::SdkClient;

/// Beta is an instance of the Agones Beta SDK
#[derive(Clone)]
pub struct Beta {
    client: SdkClient<Channel>,
}

impl Beta {

    /// new creates a new instance of the Beta SDK
    pub(crate) fn new(ch: Channel) -> Self {
        Self {
            client: SdkClient::new(ch),
        }
    }

    /// GetCounterCount returns the Count for a Counter, given the Counter's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_counter_count(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_counter(api::GetCounterRequest { name: key.to_string() })
            .await
            .map(|c| c.into_inner().count)?)
    }

    /// IncrementCounter increases a counter by the given nonnegative integer amount.
    /// Will execute the increment operation against the current CRD value. Will max at max(int64).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    /// Returns error if the count is at the current capacity (to the latest knowledge of the SDK),
    /// and no increment will occur.
    ///
    /// Note: A potential race condition here is that if count values are set from both the SDK and
    /// through the K8s API (Allocation or otherwise), since the SDK append operation back to the CRD
    /// value is batched asynchronous any value incremented past the capacity will be silently truncated.
    #[inline]
    pub async fn increment_counter(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: None,
                    count_diff: amount,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// DecrementCounter decreases the current count by the given nonnegative integer amount.
    /// The Counter Will not go below 0. Will execute the decrement operation against the current CRD value.
    /// Will error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
    #[inline]
    pub async fn decrement_counter(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: None,
                    count_diff: -amount,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// SetCounterCount sets a count to the given value. Use with care, as this will overwrite any previous
    /// invocations’ value. Cannot be greater than Capacity.
    #[inline]
    pub async fn set_counter_count(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: Some(amount.into()),
                    capacity: None,
                    count_diff: 0,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// GetCounterCapacity returns the Capacity for a Counter, given the Counter's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_counter_capacity(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_counter(api::GetCounterRequest { name: key.to_string() })
            .await
            .map(|c| c.into_inner().capacity)?)
    }

    /// SetCounterCapacity sets the capacity for the given Counter. A capacity of 0 is no capacity.
    #[inline]
    pub async fn set_counter_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: Some(amount.into()),
                    count_diff: 0,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// GetListCapacity returns the Capacity for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_capacity(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().capacity)?)
    }

    /// SetListCapacity sets the capacity for a given list. Capacity must be between 0 and 1000.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn set_list_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_list(api::UpdateListRequest {
                list: Some(api::List {
                    name: key.to_string(),
                    capacity: amount,
                    values: vec![],
                }),
                update_mask: Some(prost_types::FieldMask { paths: vec!["capacity".to_string()] }),
            })
            .await
            .map(|_| ())?)
    }

    /// ListContains returns if a string exists in a List's values list, given the List's key (name)
    /// and the string value. Search is case-sensitive.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn list_contains(&mut self, key: &str, value: &str) -> Result<bool> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values.contains(&value.to_string()))?)
    }

    /// GetListLength returns the length of the Values list for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_length(&mut self, key: &str) -> Result<usize> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values.len())?)
    }

    /// GetListValues returns the Values for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_values(&mut self, key: &str) -> Result<Vec<String>> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values)?)
    }

    /// AppendListValue appends a string to a List's values list, given the List's key (name)
    /// and the string value. Will error if the string already exists in the list.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn append_list_value(&mut self, key: &str, value: &str) -> Result<()> {
        Ok(self
            .client
            .add_list_value(api::AddListValueRequest {
                name: key.to_string(),
                value: value.to_string(),
            })
            .await
            .map(|_| ())?)
    }

    /// DeleteListValue removes a string from a List's values list, given the List's key (name)
    /// and the string value. Will error if the string does not exist in the list.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn delete_list_value(&mut self, key: &str, value: &str) -> Result<()> {
        Ok(self
            .client
            .remove_list_value(api::RemoveListValueRequest {
                name: key.to_string(),
                value: value.to_string(),
            })
            .await
            .map(|_| ())?)
    }
}
