# VEGA fees Dashboard
The aim of the project is to provide a dashboard displaying Vega network fees.

## Requirements
In order to your it locally you would need:
- A Grafana instance (cloud or selfhosted);
- A PostgreSQL instance to store data;
## How to run it
Customize the configuration file according to your env

## Grafana configuration
On Grafana, you would need to perform the following steps:
- Import the [GraphQL datasource pluging](https://grafana.com/grafana/plugins/fifemon-graphql-datasource/)
- Create a GraphQL datasource pointing to [Vega API endpoint](https://lb.testnet.vega.xyz/query)
- Create a PostgreSQL datasource pointing to your PostgreSQL instance
- Import dashboards using sources in the grafana folder of this repository 
## Architecture
 ![Infra schema](https://github.com/baldator/vega-fees/blob/master/doc/Vega-fees.png?raw=true)

## Database backup 
A daily backup is triggered by a GitHub action (dbBackup.yaml) on a daily basis. The DB backup is stored on GitHub as artifact.

## Hack.Money 2021
The project has been submitted to Hack.Money 2021 hackathon. 