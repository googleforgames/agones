using Agones.Utility;
using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using System;
using System.Collections.Generic;
using System.Net;

namespace Agones.Models
{
	public class GameServerInfoWatchResult
	{
		[JsonProperty("result")]
		public GameServerInfo GameServerInfo { get; set; }
	}

	public class GameServerInfo
	{
		[JsonProperty("object_meta")]
		public GameServerInfoMetaData MetaData { get; set; }

		[JsonProperty("status")]
		public GameServerInfoStatus Status { get; set; }
	}

	public class GameServerInfoMetaData
	{
		[JsonProperty("name")]
		public string Name { get; set; }

		[JsonProperty("namespace")]
		public string Namespace { get; set; }

		[JsonProperty("uid")]
		public string UID { get; set; }

		[JsonProperty("resource_version")]
		public string ResourceVersion { get; set; }

		[JsonProperty("creation_timestamp")]
		//[JsonConverter(typeof(UnixDateTimeConverter))] //Use this one if you dont want to support negative unix timestamps
		[JsonConverter(typeof(CustomUnixDateTimeConverter))]
		public DateTime CreationTime { get; set; }

		[JsonProperty("annotations")]
		public Dictionary<string, string> Annotations { get; set; }

		[JsonProperty("labels")]
		public Dictionary<string, string> Labels { get; set; }

		public override bool Equals(object obj)
		{
			var data = obj as GameServerInfoMetaData;

			return data != null &&
				   Name == data.Name &&
				   Namespace == data.Namespace &&
				   UID == data.UID &&
				   ResourceVersion == data.ResourceVersion &&
				   CreationTime == data.CreationTime &&
				   EqualityComparer<Dictionary<string, string>>.Default.Equals(Annotations, data.Annotations) &&
				   EqualityComparer<Dictionary<string, string>>.Default.Equals(Labels, data.Labels);
		}

		public override int GetHashCode()
		{
			var hashCode = -1011602838;
			hashCode = hashCode * -1521134295 + EqualityComparer<string>.Default.GetHashCode(Name);
			hashCode = hashCode * -1521134295 + EqualityComparer<string>.Default.GetHashCode(Namespace);
			hashCode = hashCode * -1521134295 + EqualityComparer<string>.Default.GetHashCode(UID);
			hashCode = hashCode * -1521134295 + EqualityComparer<string>.Default.GetHashCode(ResourceVersion);
			hashCode = hashCode * -1521134295 + CreationTime.GetHashCode();
			hashCode = hashCode * -1521134295 + EqualityComparer<Dictionary<string, string>>.Default.GetHashCode(Annotations);
			hashCode = hashCode * -1521134295 + EqualityComparer<Dictionary<string, string>>.Default.GetHashCode(Labels);

			return hashCode;
		}
	}

	public enum GameServerState
	{
		Scheduled,
		Ready,
		Allocated,
		Shutdown
	}

	public class GameServerInfoStatus
	{
		public class PortConfig
		{
			[JsonProperty("name")]
			public string Name { get; set; }

			[JsonProperty("port")]
			public int Port { get; set; }
		}

		[JsonProperty("state"), JsonConverter(typeof(StringEnumConverter))]
		public GameServerState State { get; set; }

		[JsonProperty("address"), JsonConverter(typeof(IPAddressJsonConverter))]
		public IPAddress Address { get; set; }

		[JsonProperty("ports")]
		public IEnumerable<PortConfig> Ports { get; set; }
	}
}