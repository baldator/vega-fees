package main

import (
	"fmt"
	"log"

	"github.com/go-pg/pg/extra/pgdebug"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

type Market struct {
	tableName struct{} `pg:"public.markets"`

	Id                 int64       `pg:"id,pk"`
	VegaId             string      `pg:"vega_id"`
	Name               string      `pg:"name"`
	Decimals           int32       `pg:"decimals"`
	Fees               []*Fee15Min `pg:"rel:has-many"`
	Offset             int64       `pg:"offset,default:0"`
	SettlementCurrency *Asset      `pg:"rel:has-one"`
	CurrencyId         string      `pg:"currency_id"`
}

type Fee15Min struct {
	tableName             struct{} `pg:"public.fees15min"`
	Id                    int64    `pg:"id,pk"`
	Market                *Market  `pg:"rel:has-one"`
	BuyInfrastructureFee  uint64   `pg:"buy_infrastructure_fee"`
	BuyMakerFee           uint64   `pg:"buy_maker_fee"`
	BuyLiquidityFee       uint64   `pg:"buy_liquidity_fee"`
	SellInfrastructureFee uint64   `pg:"sell_infrastructure_fee"`
	SellMakerFee          uint64   `pg:"sell_maker_fee"`
	SellLiquidityFee      uint64   `pg:"sell_liquidity_fee"`
	Time                  int64    `pg:"time"`
	VegaMarketID          string   `pg:"vega_market_id"`
}

type Asset struct {
	tableName struct{}  `pg:"public.asset"`
	Id        string    `pg:"id,pk"`
	Name      string    `pg:"name"`
	Symbol    string    `pg:"symbol"`
	Decimal   int32     `pg:"decimals"`
	Markets   []*Market `pg:"rel:has-many"`
}

type Pair struct {
	tableName           struct{} `pg:"public.pair"`
	Id                  string   `pg:"id,pk"`
	LastParsedTimestamp int64    `pg:"last_parsed_timestamp"`
}
type Kline struct {
	tableName struct{} `pg:"public.klines"`
	Id        int64    `pg:"id,pk"`
	Time      int64    `pg:"time"`
	Value     float64  `pg:"value"`
	PairId    string   `pg:"pair_id"`
}

// func (s Fee15Min) String() string {
// 	return fmt.Sprintf("Story<%d %s %d %v>", s.Id, s.Market.Name, s.Type, s.Value)
// }

// createSchema creates database schema for User and Story models.
func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*Market)(nil),
		(*Fee15Min)(nil),
		(*Asset)(nil),
		(*Pair)(nil),
		(*Kline)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp:        false,
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// newDBConn create a new PG connection object
func newDBConn(conf ConfigVars) (con *pg.DB) {
	address := fmt.Sprintf("%s:%d", conf.DBHost, conf.DBPort)
	options := &pg.Options{
		User:     conf.DBUser,
		Password: conf.DBPassword,
		Addr:     address,
		Database: conf.DBName,
		PoolSize: 50,
	}
	con = pg.Connect(options)
	if con == nil {
		log.Println("cannot connect to postgres")
	}

	if conf.DBDebug {
		con.AddQueryHook(pgdebug.DebugHook{
			// Print all queries.
			Verbose: true,
		})
	}
	return
}

// marketExists check if a market exists in DB
func marketExists(marketId string, db *pg.DB) bool {
	market := &Market{}
	err := db.Model(market).Where("vega_id = ?", marketId).Select()

	if err != nil {
		if err.Error() == "pg: no rows in result set" {
			return false
		} else {
			panic(err)
		}
	}

	return true
}

// createMarket insert a new market in DB
func createMarket(marketId string, name string, decimals uint64, asset string, db *pg.DB) error {
	market := &Market{
		Name:       name,
		VegaId:     marketId,
		Decimals:   int32(decimals),
		CurrencyId: asset,
	}
	_, err := db.Model(market).Insert()

	return err
}

func updateMarket(market *Market, db *pg.DB) error {
	_, err := db.Model(market).
		OnConflict("(id) DO UPDATE").
		Insert()
	return err
}

func updateFees(fee *Fee15Min, db *pg.DB) error {
	_, err := db.Model(fee).
		OnConflict("(id) DO UPDATE").
		Insert()
	return err
}

func updatePair(pair *Pair, db *pg.DB) error {
	_, err := db.Model(pair).
		OnConflict("(id) DO UPDATE").
		Insert()
	return err
}

func updateKline(kline *Kline, db *pg.DB) error {
	_, err := db.Model(kline).
		OnConflict("(id) DO UPDATE").
		Insert()
	return err
}

func updateAsset(asset *Asset, db *pg.DB) error {
	_, err := db.Model(asset).
		OnConflict("(id) DO UPDATE").
		Insert()
	return err
}

func getMarket(marketId string, db *pg.DB) (*Market, error) {
	market := &Market{}
	err := db.Model(market).Where("vega_id = ?", marketId).Select()

	if err != nil {
		if err.Error() == "pg: no rows in result set" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return market, nil
}

func getPair(pairId string, db *pg.DB) (*Pair, error) {
	pair := &Pair{}
	err := db.Model(pair).Where("id = ?", pairId).Select()

	if err != nil {
		if err.Error() == "pg: no rows in result set" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return pair, nil
}

func getFeeByTimestamp(fee *Fee15Min, db *pg.DB) (*Fee15Min, error) {
	feeTmp := &Fee15Min{}
	err := db.Model(feeTmp).Where("vega_market_id = ? AND time = ?", fee.VegaMarketID, fee.Time).Select()

	if err != nil {
		if err.Error() == "pg: no rows in result set" {
			return fee, nil
		} else {
			return &Fee15Min{}, err
		}
	}

	return feeTmp, nil
}

func getKline(kline *Kline, db *pg.DB) (*Kline, error) {
	klineTmp := &Kline{}
	err := db.Model(klineTmp).Where("pairId = ? AND time = ?", kline.PairId, kline.Time).Select()

	if err != nil {
		if err.Error() == "pg: no rows in result set" {
			return klineTmp, nil
		} else {
			return &Kline{}, err
		}
	}

	return klineTmp, nil
}
