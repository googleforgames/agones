using System;

namespace Agones.Utility
{
	public class ConsoleLogger : ILogger
	{
		public string Prefix { get; set; }

		public void Log(LogLevel level, Exception exception, string message, params object[] args)
		{
			Log(level, message, args);

			Log(LogLevel.Error, exception.ToString());
		}

		public void Log(LogLevel level, string message, params object[] args)
		{
			switch (level)
			{
				case LogLevel.Trace:
					Console.ForegroundColor = ConsoleColor.DarkGray;
					break;

				case LogLevel.Debug:
					Console.ForegroundColor = ConsoleColor.Gray;
					break;

				case LogLevel.Warning:
					Console.ForegroundColor = ConsoleColor.DarkYellow;
					break;

				case LogLevel.Error:
					Console.ForegroundColor = ConsoleColor.Red;
					break;

				case LogLevel.Info:
				default:
					Console.ForegroundColor = ConsoleColor.White;
					break;
			}

			Console.WriteLine(Prefix + message, args);

			Console.ForegroundColor = ConsoleColor.White;
		}
	}
}
