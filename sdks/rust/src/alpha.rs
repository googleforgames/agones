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

use std::sync::Arc;

use grpcio;

use super::errors::*;
use super::grpc::alpha;
use super::grpc::alpha_grpc;

/// Alpha is an instance of the Agones Alpha SDK
pub struct Alpha {
    client: Arc<alpha_grpc::SdkClient>,
}

impl Alpha {
    /// new creates a new instance of the Alpha SDK
    pub fn new(ch: grpcio::Channel) -> Alpha {
        let cli = alpha_grpc::SdkClient::new(ch);
        Alpha {
            client: Arc::new(cli),
        }
    }

    /// This returns the last player capacity that was set through the SDK.
    /// If the player capacity is set from outside the SDK, use sdk.get_gameserver() instead.
    pub fn get_player_capacity(&self) -> Result<i64> {
        let req = alpha::Empty::new();
        let count = self.client.get_player_capacity(&req).map(|c| c.count)?;
        Ok(count)
    }

    /// This changes the player capacity to a new value.
    pub fn set_player_capacity(&self, capacity: i64) -> Result<()> {
        let mut c = alpha::Count::new();
        c.set_count(capacity);
        let res = self.client.set_player_capacity(&c).map(|_| ())?;
        Ok(res)
    }

    /// This function increases the SDK’s stored player count by one, and appends this playerID to GameServer.status.players.ids.
    /// Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.
    pub fn player_connect<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self.client.player_connect(&p).map(|b| b.bool)?;
        Ok(res)
    }

    /// This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.status.players.ids.
    /// Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.
    pub fn player_disconnect<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self.client.player_disconnect(&p).map(|b| b.bool)?;
        Ok(res)
    }

    /// Returns the current player count.
    pub fn get_player_count(&self) -> Result<i64> {
        let req = alpha::Empty::new();
        let count = self.client.get_player_count(&req).map(|c| c.count)?;
        Ok(count)
    }

    /// This returns if the playerID is currently connected to the GameServer.
    /// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
    pub fn is_player_connected<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self.client.is_player_connected(&p).map(|b| b.bool)?;
        Ok(res)
    }

    /// This returns the list of the currently connected player ids.
    /// This is always accurate, even if the value has not been updated to the Game Server status yet.
    pub fn get_connected_players(&self) -> Result<Vec<String>> {
        let req = alpha::Empty::new();
        let res = self
            .client
            .get_connected_players(&req)
            .map(|pl| pl.list.into())?;
        Ok(res)
    }

    /// This returns the last player capacity that was set through the SDK.
    /// If the player capacity is set from outside the SDK, use sdk.get_gameserver() instead.
    pub async fn get_player_capacity_async(&self) -> Result<i64> {
        let req = alpha::Empty::new();
        let count = self
            .client
            .get_player_capacity_async(&req)?
            .await
            .map(|c| c.count)?;
        Ok(count)
    }

    /// This changes the player capacity to a new value.
    pub async fn set_player_capacity_async(&self, capacity: i64) -> Result<()> {
        let mut c = alpha::Count::new();
        c.set_count(capacity);
        let res = self
            .client
            .set_player_capacity_async(&c)?
            .await
            .map(|_| ())?;
        Ok(res)
    }

    /// This function increases the SDK’s stored player count by one, and appends this playerID to GameServer.status.players.ids.
    /// Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.
    pub async fn player_connect_async<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self
            .client
            .player_connect_async(&p)?
            .await
            .map(|b| b.bool)?;
        Ok(res)
    }

    /// This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.status.players.ids.
    /// Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.
    pub async fn player_disconnect_async<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self
            .client
            .player_disconnect_async(&p)?
            .await
            .map(|b| b.bool)?;
        Ok(res)
    }

    /// Returns the current player count.
    pub async fn get_player_count_async(&self) -> Result<i64> {
        let req = alpha::Empty::new();
        let count = self
            .client
            .get_player_count_async(&req)?
            .await
            .map(|c| c.count)?;
        Ok(count)
    }

    /// This returns if the playerID is currently connected to the GameServer.
    /// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
    pub async fn is_player_connected_async<S>(&self, id: S) -> Result<bool>
    where
        S: Into<String>,
    {
        let mut p = alpha::PlayerID::new();
        p.set_playerID(id.into());
        let res = self
            .client
            .is_player_connected_async(&p)?
            .await
            .map(|b| b.bool)?;
        Ok(res)
    }

    /// This returns the list of the currently connected player ids.
    /// This is always accurate, even if the value has not been updated to the Game Server status yet.
    pub async fn get_connected_players_async(&self) -> Result<Vec<String>> {
        let req = alpha::Empty::new();
        let res = self
            .client
            .get_connected_players_async(&req)?
            .await
            .map(|pl| pl.list.into())?;
        Ok(res)
    }
}

impl Clone for Alpha {
    fn clone(&self) -> Self {
        Self {
            client: Arc::clone(&self.client),
        }
    }
}
