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
These are a short description of what the api's do
+ v1/keys(/:id):
  + Post: This is where arvo_log logs your keys to
  + Get: You can get all keys for a all hosts or pass a certname to get it for a single host.
+ v1/hierarchy(/:id): This only has a get method. This either logs your hiera.yaml hierarchy or you can pass a certname to get the translated yaml locations.
+ v1/clean/(:id): This is a get method that will help you clean up hiera data. This just parses trough the keys and hiera data. 
+ v1/clean-all/refresh: this method will create the database entry for the clean-all endpoint
+ v1/clean-all: This endpoint will show all keys that were never called upon. As well as all files never read by then entries found in your log database. You first need to run the refresh endpoint. Creating the entry may take a while if you have a large environment.

## examples
### keys api
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
### clean api
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
### clean-all
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


