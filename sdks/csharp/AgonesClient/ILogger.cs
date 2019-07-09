using System;

namespace Agones.Utility
{
	public enum LogLevel
	{
		Trace,
		Debug,
		Info,
		Warning,
		Error
	}

	public interface ILogger
	{
		string Prefix { get; set; }

		void Log(LogLevel level, string message, params object[] args);

		void Log(LogLevel level, Exception exception, string message, params object[] args);
	}
}
