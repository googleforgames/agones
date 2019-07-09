using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using System;

namespace Agones.Utility
{
	public class CustomUnixDateTimeConverter : DateTimeConverterBase
	{
		static readonly DateTime EpochStartTime = new DateTime(1970, 1, 1, 0, 0, 0, DateTimeKind.Utc);

		public override object ReadJson(JsonReader reader, Type objectType, object existingValue, JsonSerializer serializer)
		{
			var milliseconds = long.Parse((string)reader.Value);

			return EpochStartTime.AddSeconds(milliseconds);
		}

		public override void WriteJson(JsonWriter writer, object value, JsonSerializer serializer)
		{
			var current = (DateTime)value;

			var sinceEpochStart = current - EpochStartTime;

			long milliseconds = (long)sinceEpochStart.TotalSeconds;

			writer.WriteRawValue(milliseconds.ToString());
		}
	}
}