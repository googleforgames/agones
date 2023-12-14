# Origin

This `README.md` was originally based on [cloud-builders-community](https://github.com/GoogleCloudPlatform/cloud-builders-community), which contains source code for community-contributed Docker images. You can use these images as build steps for [Google Cloud Build](https://cloud.google.com/build/docs). 

The key change made upon importing the [cache builder's](https://github.com/GoogleCloudPlatform/cloud-builders-community/tree/master/cache) code into Agones was the update of image tags to use Google Artifact Registry instead of Google Container Registry.

# Cache builders

This includes a pair of builders, `save_cache` and `restore_cache`, that work together to cache files between builds to a GCS bucket (or local file).

## Using the `save_cache` builder

All options that require a value use the form `--option=value` or `-o=value` so that they look nice in Yaml files.

| Option           | Description                                                 |
| ---------------- | ----------------------------------------------------------- |
| -b, --bucket     | The cloud storage bucket to upload the cache to. [optional] |
| -o, --out        | The output directory to write the cache to. [optional]      |
| -k, --key        | The cache key used for this cache file. [optional]          |
| -p, --path       | The files to store in the cache. Can be repeated.           |
| -t, --threshold  | The parallel composite upload threshold [default: 50M]      |
| -n, --no-clobber | Skips the save if the cache file already exists in GCS.     |

One of `--bucket` or `--out` parameters are required.  If `--bucket` then the cache file will be uploaded to the provided GCS bucket path.  If `--out` then the cache file will be stored in the directory specified on disk.

The key provided by `--key` is used to identify the cache file. Any other cache files for the same key will be overwritten by this one.

The `--path` parameters can be repeated for as many folders as you'd like to cache.  When restored, they will retain folder structure on disk.

The `--no-clobber` flag is used to skip creating and uploading the cache to GCS if the cache file already exists. This will shorten the time for builds when a cache was restored and is not changed by your build process. For example, this flag can be used if you are caching your dependencies and all of your dependencies are pinned to a specific version. This flag is valid only when `--bucket` is used.

## Using the `restore_cache` builder

All options use the form `--option=value` or `-o=value` so that they look nice in Yaml files.

| Option                 | Description                                                                            |
| ---------------------- | -------------------------------------------------------------------------------------- |
| -b,  --bucket          | The cloud storage bucket to download the cache from. [optional]                        |
| -s,  --src             | The local directory in which the cache is stored. [optional]                           |
| -k,  --key             | The cache key used for this cache file. [optional]                                     |
| -kf, --key_fallback    | The cache key fallback pattern to be used if exact cache key is not found. [optional]  |

One of `--bucket` or `--src` parameters are required.  If `--bucket` then the cache file will be downloaded from the provided GCS bucket path.  If `--src` then the cache file will be read from the directory specified on disk.

The key provided by `--key` is used to identify the cache file.

The fallback key pattern provide by `--key_fallback`, will be used to fetch the most recent cache file matching that pattern in case there is a cache miss from the specified `--key`.

### `checksum` Helper

As apps develop, cache needs change. For instance when dependencies are removed from a project, or versions are updated, there is no need to cache the older versions of dependencies. Therefore it's recommended that you update the cache key when these changes occur.

This builder includes a `checksum` helper script, which you can use to create a simple checksum of files in your project to use as a cache key.

To use it in the `--key` arguemnt, simply surround the command with `$()`:

```bash
--key=build-cache-$(checksum build.gradle)-$(checksum dependencies.gradle)
```

To ensure you aren't paying for storage of obsolete cache files you can add an Object Lifecycle Rule to the cache bucket to delete object older than 30 days.

## Examples

The following examples demonstrate build requests that use this builder.

### Saving a cache with checksum to GCS bucket

This `cloudbuild.yaml` saves the files and folders in the `path` arguments to a cache file in the GCS bucket `gs://$CACHE_BUCKET/`.  In this example the key will be updated, resulting in a new cache, every time the `cloudbuild.yaml` build file is changed.

```yaml
- name: 'us-docker.pkg.dev/$PROJECT_ID/ci/save_cache'
  args:
  - '--bucket=gs://$CACHE_BUCKET/'
  - '--key=resources-$( checksum cloudbuild.yaml )'
  - '--path=.cache/folder1'
  - '--path=.cache/folder2/subfolder3'
```

If your build process only changes the cache contents whenever `cloudbuild.yaml` changes, then you can skip saving the cache again if it already exists in GCS:
```yaml
- name: 'us-docker.pkg.dev/$PROJECT_ID/ci/save_cache'
  args:
  - '--bucket=gs://$CACHE_BUCKET/'
  - '--key=resources-$( checksum cloudbuild.yaml )'
  - '--path=.cache/folder1'
  - '--path=.cache/folder2/subfolder3'
  - '--no-clobber'
```

### Saving a cache with checksum to a local file

This `cloudbuild.yaml` saves the files and folders in the `path` arguments to a cache file in the directory passed to the `out` parameter.  In this example the key will be updated, resulting in a new cache, every time the `cloudbuild.yaml` build file is changed.

```yaml
- name: 'us-docker.pkg.dev/$PROJECT_ID/ci/save_cache'
  args:
  - '--out=/cache/'
  - '--key=resources-$( checksum cloudbuild.yaml )'
  - '--path=.cache/folder1'
  - '--path=.cache/folder2/subfolder3'
  volumes:
  - name: 'cache'
    path: '/cache'
```

### Restore a cache from a GCS bucket

This `cloudbuild.yaml` restores the files from the compressed cache file identified by `key` on the cache bucket provided, if it exists.

```yaml
- name: 'us-docker.pkg.dev/$PROJECT_ID/ci/restore_cache'
  args:
  - '--bucket=gs://$CACHE_BUCKET/'
  - '--key=resources-$( checksum cloudbuild.yaml )'
```

### Restore a cache from a local file

This `cloudbuild.yaml` restores the files from the compressed cache file identified by `key` on the local filesystem, if it exists.

```yaml
- name: 'us-docker.pkg.dev/$PROJECT_ID/ci/restore_cache'
  args:
  - '--src=/cache/'
  - '--key=resources-$( checksum cloudbuild.yaml )'
  volumes:
  - name: 'cache'
    path: '/cache'
```

### Restore a cache with a fallback key

```yaml
- name: us-docker.pkg.dev/$PROJECT_ID/ci/restore_cache
  id: restore_cache
  args: [
    '--bucket=gs://${_CACHE_BUCKET}',
    '--key=gradle-$( checksum checksum.txt )',
    '--key_fallback=gradle-',
  ]
```
