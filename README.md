# arvo
Arvo is a hiera helping tool for puppet. It needs to be used in combination with https://github.com/Bryxxit/arvo_log. Arvo uses both puppetdb and mongodb. Mongodb as a database to store the logged hiera keys. 

Puppetdb is used to collect facts of a node. The facts are needed to translates paths in your hierarchy. As for now arvo can only read from hiera files. Further improvements may be added flater.  

The default location of arvo is http://localhost:8162

# Configuration
## command line
On the command line you can configure two parmaters. The port it is running on and the location of the configuration file.
```
arvo -listen-address 0.0.0.0 -port 8162 -swagger-host localhost -conf arvo.yaml
```
+ listen-address: This is the address the application will listen on. If you want to limit it by ip and such.
+ port: The port the application will run on by default 8162
+ swagger-host: Is the hostname that appears in the swagger documentation
+ conf: The location of the config file

## Configuration file
The configuration file is a yaml based file following things can be configured.
```
puppet:
  host: localhost
  port: 8080
  ssl: false
  key: ""
  ca: ""
  cert: ""
  insecure: false
db:
  host: localhost
  port: 27017
  db: arvo
  password: ""
  username: ""
key_ttl_minutes: 15
datadir: "/etc/puppetlabs/code/environment/production/data"
hiera_file: "/etc/puppetlabs/puppet/hiera.yaml"
```
+ puppet: Contains connection info to your puppetdb instance. By default ssl is disabled. You can however configure it.
+ db: Contains data for your mongodb connection. For auth you'll need to provider user/pass
+ key_ttl_minutes: This is the time to keep logged hiera keys for in minutes. So when the next keys logs all logs older than this value will be removed.
+ datadir: The location of your hiera data.
+ hiera_file: The location of the hiera.yaml file so where your hierarchies are defined.

# Api
We have now integrated swagger into the project and it should be available at: http://localhost:8162/swagger/index.html

We have two major sections of the api one being the cleanup helper section the other being the hiera management section
## Cleanup
These api endpoint are there to log hiera lookup keys to and then you can use this information to call the other endpoints. These will give you a general idea
+ v1/keys(/:id):
  + Post: This is where arvo_log logs your keys to
  + Get: You can get all keys for a all hosts or pass a certname to get it for a single host.
+ v1/hierarchy(/:id): This only has a get method. This either logs your hiera.yaml hierarchy or you can pass a certname to get the translated yaml locations.
+ v1/clean/(:id): This is a get method that will help you clean up hiera data. This just parses trough the keys and hiera data. 
+ v1/clean-all/refresh: this method will create the database entry for the clean-all endpoint
+ v1/clean-all: This endpoint will show all keys that were never called upon. As well as all files never read by then entries found in your log database. You first need to run the refresh endpoint. Creating the entry may take a while if you have a large environment.

### examples
#### keys api
```
curl localhost:8162/v1/keys/certname
{
  "id": "certname",
  "keys": [
    {
      "certname": "certname",
      "key": "firewalls",
      "date_string": "2020-05-18T08:37:29+0200"
    }
.....
```
#### clean api
```
curl localhost:8162/v1/clean/certname
{
  "in_log_not_in_hiera": ["array", "of", "keys"],
  "in_log_and_hiera": [
                      {
                        "key": "keyname",
                        "paths": ["array", "of", "locations"]
                      }
                      ],
  "in_hiera_not_in_log": [
                         {
                           "key": "keyname",
                           "paths": ["array", "of", "locations"]
                         }
                         ],
  "duplicates":[
               {
                 "key": "keyname",
                 "paths": ["array", "of", "locations"]
               }
               ]
}
```
+ in log not in hiera: These are keys that were called upon by hiera but have not been found in any of the
files. This may sometimes come in handy if you're debugging lookups etc.
+ in log and hiera: These are keys that are found in hiera and in the log. This can be handy to see if keys
are declared multiple times. And thus are overwritten/merged.
+ in hiera not in log: These keys were found in hiera but were never called upon by lookup. This can be
useful mostly on the node and in a lesser amount platform level to see which keys can
safetly be removed.
+ duplicate: At the moment this does not work for hashes but it will for the rest of the data and
hashes are in the works. This is useful because you can clean up this data and save some disk space or more specific files.
#### clean-all
```
curl localhost:8162/v1/clean-all
{
    "id": "full",
    "paths_never_used": [
        "/hieradata/localhost.yaml",

    ],
    "keys_never_used": [
        {
            "paths": [
                "/hieradata/os/RedHat.yaml",
                "/hieradata/common.yaml"
            ],
            "key": "test::key"
        }
    ]
}
```
+ paths never used: Are files that are present in your hiera data but are never called upon. These can be removed if they're not going to be used in the near future?
+ keys never used: This time we got to all entries in the database and see which keys are not used. These keys did not appear in any of the logs and can thus be removed.

## Hiera 
This section is there if you want to manage hiera with arvo. The hiera works similar to hiera files were you just have a path were your values are stored.
For example common. yaml would be hiera/path/common. However what arvo offers extra is variables that can be used similar to a hiera hierachy.
So in the config file you can also define a hierachy. Puppet facts can also be used here. 
```
---
datadir: /datadir/"
hiera_file: "hiera.yaml"
hierarchy:
  - "%{hostname}"
  - "%{os.family}-%{operatingsystemmajrelease}"
  - "%{os.family}"
  - "%{environment}"
  - "common"
```
This hierarchy will also be translated when a node does a call to the actual values and will retrieve the first value from the hierarchy. 
Variables can also be set in the hiera data by using ${arvo::var_name}. These variables are set in the variable part of the hiera api.

#### endpoints:
+ v1/hiera/path(/:id): GET/POST/PUT/DELETE This endpoint allows you to manage hiera values with arvo variables on a specified key. 
+ v1/hiera/variable/hierarchy(/:id): GET This endpoint returns the hierarchy for variables defined inside the config. If you pass a certname you'll get the hieracht with the facts replaced by its values.
+ v1/hiera/variable/path(/:id): GET/POST/PUT/DELETE This endpoint will allow you to set variable values on a specfied hierarchy path. 
+ v1/hiera/value/:id/:certname: GET This endpoint gets the hiera values for a specified hiera key. You must also provide a certname as the arvo variables will be retrieved and replaced in the values.

#### example
We have set a hiera path with one of our arvo variables in it
```
curl -X GET "http://localhost:8162/v1/hiera/path/common" -H "accept: application/json"
-------
{
  "_id": "common",
  "service::ensure": "stopped",
  "packages": [
    "net-tools"
  ],
  "test": "${arvo::test}"
}
```

Next we'll set this vairable in one of our arvo variable hierarchy.
```
curl -X POST "http://localhost:8162/v1/hiera/variable/path/common" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"test\": \"string from variables\"}"
-------
{
  "message": "Inserted one entry id: common",
  "success": true
}
```

Lastly we'll get the actual values with the variables translated

```
curl -X GET "http://localhost:8162/v1/hiera/value/common/somehostname" -H "accept: application/json"
-------
{
  "_id": "common",
  "service::ensure": "stopped",
  "packages": [
    "net-tools"
  ],
  "test": "string from variables"
}
```

One final note for now seems to be that because you need to retrieve hierarchy/facts and translate for a node. The speed to resolve these variables are not super fast.
Maybe in the future I should look into resolving beforehand and saving to the database and when called for I could return the result faster. This might however result in a much larger dataset.
As we would need to storer a whole dataset for each node/path combo. 