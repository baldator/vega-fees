# VEGA fees Dashboard
The aim of the project is to provide a dashboard displaying Vega network fees.

## Requirements
In order to your it locally you would need:
- A Grafana instance (cloud or selfhosted);
- A PostgreSQL instance to store data;
## How to run it
Customize the configuration file according to your environment. 
### Configuration file

### Command line parameters

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

# About Vega
[Vega](https://vega.xyz) is a protocol for creating and trading derivatives on a fully decentralised network. The network, secured with proof-of-stake, will facilitate fully automated, end-to-end margin trading and execution of complex financial products. Anyone will be able to build decentralised markets using the protocol.

Read more at [https://vega.xyz](https://vega.xyz).

# Hack.Money 2021
 ![hackmoney logo](https://github.com/baldator/vega-fees/blob/master/doc/hackMoney.png?raw=true)
The project has been submitted to Hack.Money 2021 hackathon. 