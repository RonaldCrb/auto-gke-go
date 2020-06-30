# Auto-GKE

create a cluster, deploy a little demo app with a loadbalancer using our favorite language GO!

## REQUIREMENTS

1. Go 1.13 or higher
2. Pulumi CLI
3. the Google cloud SDK (cli)
4. a project created in GCP with billing enabled 

## USAGE

first you need to login in GCP with:
```
gcloud auth application-default login
```

then make sure you are logged in with 
```
pulumi login
```

then create a new stack
```
pulumi stack init
```

then configure the 2 required variables
```
pulumi config set gcp:project yourProject
pulumi config set gcp:zone yourZone
```

fire it up!
```
pulumi up
```

to tear it down
```
pulumi destroy
```