## TODO:
Write some more about this

Config Example
```
environment: DEV
repository:
  url: git@github.com:patrickfnielsen/gitops.git
  branch: main
  ssh_key_file: ./id_ed25519
  override_identifier: server1
deployment:
  directory: ./deployments
```