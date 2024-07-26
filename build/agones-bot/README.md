# agones-bot Github Notification Bot

This is the bot that created comments on GitHub pull requests whenever a
[Google Cloud Build](https://cloud.google.com/build) build passes or fails.

## Setup

### Image

Run `make build` to submit the cloud build to create the image that will be hosted on
[Cloud Run](https://cloud.google.com/run).

### Secrets

It is expected that there will be a secret named `agones-bot-pr-commenter` with a 
[Github auth token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) stored
in it, and `agones-bot-pr-commenter@agones-images.iam.gserviceaccount.com` has the
role `roles/secretmanager.secretAccessor` for the `gh-token` secret.

### Deployment

Run `make deploy` to copy the config to the appropriate bucket, and deploy the notifier image to Cloud Run.

### Connect to Cloud Build

Follow https://cloud.google.com/build/docs/subscribe-build-notifications to create the
[Google Cloud PubSub topics](https://cloud.google.com/pubsub) for Google Cloud Build. 

Follow https://cloud.google.com/run/docs/triggering/pubsub-push to setup the Notifier service to be triggered from
PubSub with the `cloud-builds` pubsub topic.
