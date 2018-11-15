cmake -G "Visual Studio 15 2017" & MSBuild agonessdk.sln /m /t:Build /p:Configuration=Debug & MSBuild agonessdk.sln /m /t:Build /p:Configuration=Release
