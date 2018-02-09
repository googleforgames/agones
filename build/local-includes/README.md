# Local includes

You can drop files in this folder to override default build variables.

For instance if you're behind a corporate proxy you could do this :

```
cat << 'EOF' > docker-build-args.mk 
DOCKER_BUILD_ARGS = --build-arg HTTP_PROXY=$(HTTP_PROXY) --build-arg HTTPS_PROXY=$(HTTPS_PROXY) \
                    --build-arg NO_PROXY=$(NO_PROXY) --build-arg http_proxy=$(HTTP_PROXY) \
                    --build-arg https_proxy=$(HTTPS_PROXY) --build-arg no_proxy=$(NO_PROXY)
EOF
```

And all `docker build` commands in the makefile will use your proxy.