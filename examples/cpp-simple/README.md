# Simple C++ Example

This is a very simple "server" that doesn't do much other than show how the SDK works in C++.

It will
- Setup the Agones SDK
- Call `SDK::Ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log saying "Hi! I'm a Game Server"
- After 60 seconds, call `SDK::Shutdown()` to shut the server down.

## Building
Depends on the [`sdks/cpp`](../../sdks/cpp) SDK and its dependencies have been compiled and installed. 