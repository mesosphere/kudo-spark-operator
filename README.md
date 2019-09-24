# KUDO Spark Operator

# Developing

### Prerequisites

Required software:
* Docker
* GNU Make 4.2.1 or higher
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

For test cluster provisioning and Stub Universe artifacts upload valid AWS access credentials required:
* `AWS_PROFILE` **or** `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables should be provided

For pulling private repos, a GitHub token is required:
* generate [GitHub token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) 
and export environment variable with token contents: `export GITHUB_TOKEN=<your token>`
  * or save the token either to `<repo root>/shared/data-services-kudo/.github_token` or to `~/.ds_kudo_github_token` 

### Build steps

GNU Make is used as the main build tool and includes the following main targets:
* `make cluster-create-[konvoy|mke]` creates a Konvoy or MKE cluster
* `make cluster-destroy-[konvoy|mke]` creates a Konvoy or MKE cluster
* `make cluster-destroy-all` destroys all clusters created by `make cluster-create-[konvoy|mke]`
* `make clean-all` removes all artifacts produced by targets from local filesystem

A typical workflow looks as following:
```
make clean-all
make cluster-create
make test
make cluster-destroy
```