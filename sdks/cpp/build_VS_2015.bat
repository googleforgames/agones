cmake -G "Visual Studio 14 2015" & MSBuild agonessdk.sln /m /t:Build /p:Configuration=Debug & MSBuild agonessdk.sln /m /t:Build /p:Configuration=Release
