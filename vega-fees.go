package main

import (
	"context"
	"flag"
	"log"
	"math"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/vegaprotocol/api/grpc/clients/go/generated/code.vegaprotocol.io/vega/proto/api"
	"google.golang.org/grpc"
)

const offsetPagination = 100

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

	if *klinesUpdate {
		err = getBinanceKlines(db)
		if err != nil {
			panic(err)
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
				err = createMarket(element.Id, element.TradableInstrument.Instrument.Name, element.DecimalPlaces, element.TradableInstrument.Instrument.GetFuture().SettlementAsset, db)
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

func getBinanceKlines(db *pg.DB) error {

	return nil
}
