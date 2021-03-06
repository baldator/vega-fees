# VEGA fees Dashboard
The aim of the project is to provide a dashboard displaying Vega network fees.

## Requirements
In order to your it locally you would need:
- A Grafana instance (cloud or selfhosted);
- A PostgreSQL instance to store data;
## How to run it
Customize the configuration file according to your environment. 
### Configuration file
Here is an example configuration file:
```yaml
# Vega parameters
GrpcNodeUrl: "n07.testnet.vega.xyz:3002"
Debug: true # if true will log all query before executing them 

# DB parameters
DBHost: "MyDBHost"
DBUser: "MyDBUser"
DBPassword: "MyDBPassword"
DBName: "MyDBName"
DBdebug: true

# Klines parameters
Pairs: 
  - BTCUSDT
  - EURUSDT
  - USDTDAI
  - USDCUSDT
```
### Command line parameters
The script is modular and functions can be selected using command line parameters. Available features are: 
- createDB: creates the database schema;
- feesUpdate: scrapes fees using Vega gRPC API;
- klinesUpdate: scrapes assets to USD conversion rates from Binance;
- candleUpdate: scrapes 15min candles using Vega gRPC API;

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