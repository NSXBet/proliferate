
Usage

```bash
# Enter the environment
docker compose run --rm sh

# Apply the patch
python main.py apply_patch patches/patch-01-add-metadata.py -c test/context.yaml
```

# Setting up with 1Password CLI

1. Make sure you're logged into 1Password CLI:

```bash
# Ensure .env file exists
touch .env

# Get GitHub token from 1Password and write to .env
echo "GITHUB_TOKEN=$(op item get "tidbits-masspr-github-token" --fields "password" --reveal)" > .env
```



# TODO

- [ ] Make assignees work
- [ ] Make status print from the output of every patch