# GEAR
**What is GEAR?**

It stands for `Git-Enabled Automation and Release` and it's a very basic GitOps system based on docker-compose files.

**Why do we need this?**

If running a full blown kubernetes cluster is to much, but you still want the benefits of GitOps, this could be a solution. While it's still in very early stages, it does work.

**How does it work?**

It works by checking a remote git repository for changes on a fixed internval (default 1 minute). When it detects changes it pull's down the repository, and saves all `*.yaml` files in something it calls a `Bundle`.

It then takes the bundle and tries to startup the files from it using docker compose. It's possible to have a "base" docker-compose file, and an override - This might be usefull in case you have multiple servers that requires the same services, but with different config.

To do this create a file named `<something.yaml>` and then create a dirctory named `customise` in that directory you then create a sub-directory using the `override_identifier` from the config, and last you create a override file named `<something.yaml>`.

Here's an example structure where the `override_identifier` is set to *server1*
```
/something.yaml
/customise/server1/something.yaml
```
**Note:** This follows the standard docker-compose override, see that for documentation on how to override things.

**What about secrets?**

It's possible to encrypt files using [age](https://github.com/FiloSottile/age/tree/main), currently only SSH keys are supported.
While we don't recommend it, it's possible to use the same SSH key that's used to access the repository.

The private key needs to be specified using `encryption_key_file`.

When encryption is enabled, the git sync engine will decrypt files before it starts the deploy process. Due to the native of the system, all files will be stored unencrypted on the disk afterwards. The reasoning about this is that they key is also stored on the same host.

### Encryption Example
  1) Generate a new ssh key pair using:
     - `ssh-keygen -t ed25519 -f id_ed25519 -c "git-encryption"`
  2) Encrypt the file using the public key:
     - `age -R id_ed25519.pub example.yaml > example.yaml.enc`
  3) Check-in the encrypted file into the repository, make sure to not check-in the none encrypted file

## Config Example
```
environment: DEV
sync_interval: 60 # seconds between checks
encryption_key_file: ./id_ed25519-enc
repository:
  url: git@github.com:patrickfnielsen/gitops.git
  branch: main
  ssh_key_file: ./id_ed25519
  override_identifier: server1
deployment:
  directory: ./deployments
```