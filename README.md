# Auto semver tag

Action creates semver tag *(vX.Y.Z)* when pull request is merged. New version is calculated based on PR labels: "major", "minor". Patch version is incremented by default.

## Enviroment

### `GITHUB_TOKEN` **(required)**

[PersonalAccessToken](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token) with repo scope.

## Inputs

### `release_branch` **(required)**
Branch to tag.

## Example

```yaml
# .github/workflows/auto-semver-tag.yml
name: auto-semver-tag

on:
  pull_request_target:
    types: [ closed ]

jobs:
  tagging:
    runs-on: ubuntu-latest
    steps:
      - name: auto-semver-tag
        uses: infobloxopen/auto-semver-tag@master
        with:
          release_branch: master
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

