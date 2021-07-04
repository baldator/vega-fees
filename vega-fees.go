package main

import (
	"context"
	"flag"
	"log"
	"math"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/vegaprotocol/api/grpc/clients/go/generated/code.vegaprotocol.io/vega/proto"
	"github.com/vegaprotocol/api/grpc/clients/go/generated/code.vegaprotocol.io/vega/proto/api"
	"google.golang.org/grpc"
)

const offsetPagination = 100
const binanceImportFrom = 1622498400

func main() {
	// Read application config
	conf, err := ReadConfig("config.yaml")
	if err != nil {
		log.Fatal("Failed to read config: ", err)
	}

	// Parse command line parameters
	createDB := flag.Bool("createDB", false, "Create the DB structure based on the ORM structure.")
	feesUpdate := flag.Bool("feesUpdate", false, "Import fees from Vega gRPC API.")
	klinesUpdate := flag.Bool("klinesUpdate", false, "Import crypto market historical data from Binance")
	candleUpdate := flag.Bool("candleUpdate", false, "Import market historical data from Vega")
	testFlag := flag.Bool("testFlag", false, "Import market historical data from Vega")
	flag.Parse()

	// Initialize DB
	log.Println("Initialize DB connection")
	db := newDBConn(conf)

	// Create DB
	if *createDB {
		log.Println("Create DB schema")
		err := createSchema(db)
		if err != nil {
			panic(err)
		}
	}

	conn, err := grpc.Dial(conf.GrpcNodeURL, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	dataClient := api.NewTradingDataServiceClient(conn)

	// Update vega assets
	log.Println("Update assets...")
	err = updateAssets(dataClient, db)
	if err != nil {
		panic(err)
	}

	// Request a list of markets available on the specified Vega Network
	request := api.MarketsRequest{}
	markets, err := dataClient.Markets(context.Background(), &request)
	if err != nil {
		panic(err)
	}

	for _, element := range markets.Markets {
		// Check if market already exist
		log.Printf("Update Market: %s, %s...\n", element.Id, element.TradableInstrument.Instrument.Name)
		if !marketExists(element.Id, db) {
			err = createMarket(element.Id, element.TradableInstrument.Instrument.Name, element.DecimalPlaces, element.TradableInstrument.Instrument.GetFuture().SettlementAsset, element.Fees.Factors.InfrastructureFee, element.Fees.Factors.LiquidityFee, element.Fees.Factors.MakerFee, db)
			if err != nil {
				panic(err)
			}

		}
	}

	if *klinesUpdate {

		log.Printf("Klines update: %v\n", klinesUpdate)
		err = getBinanceKlines(conf.Pairs, db)
		if err != nil {
			panic(err)
		}
	}

	if *candleUpdate {
		for _, element := range markets.Markets {
			err = candleUpdateForMarket(element.Id, dataClient, db)
			if err != nil {
				panic(err)
			}
		}
	}

	if *testFlag {
		for _, element := range markets.Markets {
			if element.TradableInstrument.Instrument.Name == "Tesla Quarterly (31 Dec 2021)" {
				market, err := getMarket(element.Id, db)
				if err != nil {
					panic(err)
				}
				marketID := market.VegaId
				offset := market.Offset
				pagination := api.Pagination{
					Descending: false,
					Skip:       uint64(offset),
					Limit:      offsetPagination,
				}
				tradesByMarketReq := api.TradesByMarketRequest{MarketId: marketID, Pagination: &pagination}
				tradesByMarketResp, err := dataClient.TradesByMarket(context.Background(), &tradesByMarketReq)

				for _, trade := range tradesByMarketResp.Trades {
					log.Printf("-- %+v\n", trade)
				}
			}
		}
	}

	if *feesUpdate {

		for _, element := range markets.Markets {
			// index is the index where we are
			// element is the element from someSlice for where we are

			//fmt.Printf("Market[%d]: %+v \n\n", index, element)
			log.Printf("Importing fees for market: %+s \n\n", element.TradableInstrument.Instrument.Name)

			// Check if market already exist
			if !marketExists(element.Id, db) {
				err = createMarket(element.Id, element.TradableInstrument.Instrument.Name, element.DecimalPlaces, element.TradableInstrument.Instrument.GetFuture().SettlementAsset, element.Fees.Factors.InfrastructureFee, element.Fees.Factors.LiquidityFee, element.Fees.Factors.MakerFee, db)
				if err != nil {
					panic(err)
				}

			}

			market, err := getMarket(element.Id, db)
			if err != nil {
				panic(err)
			}

			exitLoop := false
			offset := market.Offset

			var tmpFee = &Fee15Min{}
			marketID := market.VegaId

			if market.Fees != nil {
				tmpFee = market.Fees[len(market.Fees)-1]
			} else {
				tmpFee.Time = 0
				tmpFee.Id = 0
				tmpFee.SellInfrastructureFee = 0
				tmpFee.SellLiquidityFee = 0
				tmpFee.SellMakerFee = 0
				tmpFee.BuyInfrastructureFee = 0
				tmpFee.BuyLiquidityFee = 0
				tmpFee.BuyMakerFee = 0
				tmpFee.VegaMarketID = element.Id
			}

			for !exitLoop {
				pagination := api.Pagination{
					Descending: false,
					Skip:       uint64(offset),
					Limit:      offsetPagination,
				}

				tradesByMarketReq := api.TradesByMarketRequest{MarketId: marketID, Pagination: &pagination}
				tradesByMarketResp, err := dataClient.TradesByMarket(context.Background(), &tradesByMarketReq)

				if err != nil {
					break
				}

				for index, trade := range tradesByMarketResp.Trades {
					tradeTime := floor15minutes(trade.Timestamp)
					if tradeTime != tmpFee.Time {

						// if tmpFee is not empty store it to DB
						if tmpFee.Time > 0 {
							log.Printf("Add fee to DB. Timestamp: %d. New timeframe: %d\n", trade.Timestamp, tradeTime)

							prevFee, err := getFeeByTimestamp(tmpFee, db)
							if err != nil {
								panic(err)
							}
							if prevFee != nil {
								tmpFee.Id = prevFee.Id
								tmpFee.SellInfrastructureFee = tmpFee.SellInfrastructureFee + prevFee.SellInfrastructureFee
								tmpFee.SellLiquidityFee = tmpFee.SellLiquidityFee + prevFee.SellLiquidityFee
								tmpFee.SellMakerFee = tmpFee.SellMakerFee + prevFee.SellMakerFee
								tmpFee.BuyInfrastructureFee = tmpFee.BuyInfrastructureFee + prevFee.BuyInfrastructureFee
								tmpFee.BuyLiquidityFee = tmpFee.BuyLiquidityFee + prevFee.BuyLiquidityFee
								tmpFee.BuyMakerFee = tmpFee.BuyMakerFee + prevFee.BuyMakerFee
							}

							err = updateFees(tmpFee, db)
							if err != nil {
								panic(err)
							}

							market.Offset = offset + int64(index)
							err = updateMarket(market, db)
							if err != nil {
								panic(err)
							}

						}

						// check if fees already exists
						tmpFee.BuyInfrastructureFee = 0
						tmpFee.BuyMakerFee = 0
						tmpFee.BuyLiquidityFee = 0
						tmpFee.SellInfrastructureFee = 0
						tmpFee.SellMakerFee = 0
						tmpFee.SellLiquidityFee = 0
						tmpFee.Time = tradeTime
						tmpFee.VegaMarketID = element.Id

						tmpFee.Id = 0
					}

					// log.Println(trade)
					if tradeTime == tmpFee.Time {
						if trade.BuyerFee != nil {
							tmpFee.BuyInfrastructureFee = tmpFee.BuyInfrastructureFee + trade.BuyerFee.InfrastructureFee
							tmpFee.BuyLiquidityFee = tmpFee.BuyLiquidityFee + trade.BuyerFee.LiquidityFee
							tmpFee.BuyMakerFee = tmpFee.BuyMakerFee + trade.BuyerFee.MakerFee
						}
						if trade.SellerFee != nil {
							tmpFee.SellInfrastructureFee = tmpFee.SellInfrastructureFee + trade.SellerFee.InfrastructureFee
							tmpFee.SellLiquidityFee = tmpFee.SellLiquidityFee + trade.SellerFee.LiquidityFee
							tmpFee.SellMakerFee = tmpFee.SellMakerFee + trade.SellerFee.MakerFee
						}
					}
				}

				if len(tradesByMarketResp.Trades) < offsetPagination {
					exitLoop = true
					// save data before exit
					log.Printf("Less than %d trades: exiting, last timestamp: %d. Save data to DB\n", len(tradesByMarketResp.Trades), tradesByMarketResp.Trades[len(tradesByMarketResp.Trades)-1].Timestamp)
					prevFee, err := getFeeByTimestamp(tmpFee, db)
					if err != nil {
						panic(err)
					}
					if prevFee != nil {
						tmpFee.Id = prevFee.Id
						tmpFee.SellInfrastructureFee = tmpFee.SellInfrastructureFee + prevFee.SellInfrastructureFee
						tmpFee.SellLiquidityFee = tmpFee.SellLiquidityFee + prevFee.SellLiquidityFee
						tmpFee.SellMakerFee = tmpFee.SellMakerFee + prevFee.SellMakerFee
						tmpFee.BuyInfrastructureFee = tmpFee.BuyInfrastructureFee + prevFee.BuyInfrastructureFee
						tmpFee.BuyLiquidityFee = tmpFee.BuyLiquidityFee + prevFee.BuyLiquidityFee
						tmpFee.BuyMakerFee = tmpFee.BuyMakerFee + prevFee.BuyMakerFee
					}
					err = updateFees(tmpFee, db)
					if err != nil {
						panic(err)
					}

					market.Offset = offset + int64(len(tradesByMarketResp.Trades))
					err = updateMarket(market, db)
					if err != nil {
						panic(err)
					}
				} else {
					offset = offset + offsetPagination
					log.Printf("Adding %d to offset. Current offset is: %d\n", offsetPagination, offset)
				}

			}

		}
	}

	// Close DB connection
	db.Close()
}

func floor15minutes(timestamp int64) int64 {
	tm := time.Unix(timestamp/int64(time.Second), 0)
	minutes := int(math.Floor(float64(tm.Minute()/15)) * 15)

	t1 := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), minutes, 0, 0, tm.Location())

	return t1.Unix()
}

func updateAssets(dataClient api.TradingDataServiceClient, db *pg.DB) error {
	request := api.AssetsRequest{}
	assets, err := dataClient.Assets(context.Background(), &request)
	if err != nil {
		return err
	}

	for _, asset := range assets.Assets {
		tmpAsset := &Asset{
			Id:      asset.Id,
			Name:    asset.Details.Name,
			Symbol:  asset.Details.Symbol,
			Decimal: int32(asset.Details.Decimals),
		}
		err := updateAsset(tmpAsset, db)
		if err != nil {
			return err
		}
	}

	return nil
}

func getBinanceKlines(pairs []string, db *pg.DB) error {
	for _, pair := range pairs {
		log.Printf("Get klines for pair %s\n", pair)
		tmpPair, err := getPair(pair, db)
		if err != nil {
			return err
		}

		// Create pair in DB if it doesn't exist
		if tmpPair == nil {
			tmpPair = &Pair{
				Id:                  pair,
				LastParsedTimestamp: binanceImportFrom,
			}

			err = updatePair(tmpPair, db)
			if err != nil {
				return err
			}

			tmpPair, err = getPair(pair, db)
			if err != nil {
				return err
			}
		}

		err = scrapeKlines(tmpPair, db)
		if err != nil {
			return err
		}
	}

	return nil
}

func candleUpdateForMarket(market string, dataClient api.TradingDataServiceClient, db *pg.DB) error {
	lastTimestamp, err := getLastCandleTimestamp(market, db)
	if err != nil {
		return err
	}
	log.Println("Getting candles from Vega")
	request := api.CandlesRequest{MarketId: market, Interval: proto.Interval_INTERVAL_I15M, SinceTimestamp: lastTimestamp}
	candles, err := dataClient.Candles(context.Background(), &request)
	if err != nil {
		return err
	}

	log.Println("Parsing candles")
	for _, candle := range candles.Candles {
		err = updateCandle(&Candle{
			Time:         candle.Timestamp,
			Open:         int64(candle.Open),
			Volume:       int64(candle.Volume),
			VegaMarketID: market,
		}, db)
		if err != nil {
			return err
		}
	}

	return nil
}
