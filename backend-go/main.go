package main

import (
	"cinema/component/appctx"
	"cinema/docs"
	"cinema/middleware"
	ginauditorium "cinema/module/auditorium/transport/gin"
	gincinema "cinema/module/cinema/transport/gin"
	gincompany "cinema/module/company/transport/gin"
	gingenre "cinema/module/genre/transport/gin"
	ginmovie "cinema/module/movie/transport/gin"
	ginmoviesgenres "cinema/module/movies_genres/transport/gin"
	ginshow "cinema/module/show/transport/gin"
	ginticket "cinema/module/ticket/transport/gin"
	"cinema/module/user/transport/ginuser"
	"cinema/module/user/userstore"
	"cinema/plugin/caching"
	"cinema/plugin/caching/sdkredis"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"time"
)

func startTracing() (*trace.TracerProvider, error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint("jaeger:4318"),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("cinema-app"),
			),
		),
	)

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider, nil
}

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	dsn := "host=" + postgresHost +
		" user=" + postgresUser +
		" password=" + postgresPassword +
		" dbname=" + postgresDB +
		" port=5432 sslmode=disable"

	//content, err := os.ReadFile("local_db_env")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	// dsn = strings.ReplaceAll(string(content), "\n", " ")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	{
		i := 0
		for err != nil {
			i++
			if i == 4 {
				panic(err)
			}
			log.Println(err)
			// wait for 5 seconds before trying to connect to the database again
			<-time.After(5 * time.Second)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		}
	}

	// Set up the database connection pool
	configSQLDriver, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	configSQLDriver.SetMaxIdleConns(300)
	configSQLDriver.SetMaxOpenConns(380) // kept a few connections as buffer for developers
	configSQLDriver.SetConnMaxIdleTime(30 * time.Minute)
	configSQLDriver.SetConnMaxLifetime(time.Hour)

	db = db.Debug()
	redisDB := sdkredis.NewRedisDB(
		"MyRedis",
		&sdkredis.RedisDBOpt{
			RedisUri:  fmt.Sprintf("redis://%s:6379", os.Getenv("REDIS_HOST")),
			MaxActive: sdkredis.DefaultRedisMaxActive,
			MaxIde:    sdkredis.DefaultRedisMaxIdle,
		})

	// Configure and run the Redis client
	if err := redisDB.Run(); err != nil {
		log.Fatalf("Failed to configure Redis: %v", err)
	}
	defer func() {
		<-redisDB.Stop()
	}()
	client := redisDB.Get().(*redis.Client)
	if client == nil {
		log.Fatalln("Failed to get Redis client")
	}

	tracerProvider, err := startTracing()
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shutdown TracerProvider: %v", err)
		}
	}()

	tracer := otel.GetTracerProvider().Tracer("")
	key := os.Getenv("SECRET_KEY")
	appCtx := appctx.NewAppContext(db, key, client, &tracer)

	r := gin.Default()
	r.Use(
		otelgin.Middleware("", otelgin.WithTracerProvider(tracerProvider)),
		middleware.AddTrace(tracer),
		middleware.Recover(appCtx),
		middleware.CORSMiddleware(appCtx),
	)

	docs.SwaggerInfo.BasePath = "/v1"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	v1 := r.Group("/v1")

	userStore := userstore.NewSQLStore(db)
	cacheStore := sdkredis.NewRedisCache(appCtx)
	userCaching := caching.NewUserCaching(cacheStore, userStore)

	{
		cinemas := v1.Group("/cinemas")
		//GET /v1/cinemas
		cinemas.GET("", gincinema.ListCinema(appCtx))

		//GET /v1/cinemas/name/:name
		cinemas.GET("/name/:name", gincinema.GetCinemaWithName(appCtx))

		//GET /v1/cinemas/:id
		cinemas.GET("/:id", gincinema.GetCinemaWithID(appCtx))

		//POST /v1/cinemas
		cinemas.POST("",
			middleware.RequireAuth(appCtx, userCaching),
			middleware.CheckRole(appCtx, "admin", "cinema_owner"),
			gincinema.CreateCinema(appCtx))

		//GET /v1/cinemas/:id/auditoriums
		cinemas.GET("/:id/auditoriums", ginauditorium.ListAuditoriumWithCinemaID(appCtx))

		//GET /v1/cinemas/name/:name/auditoriums
		cinemas.GET("/name/:name/auditoriums", ginauditorium.ListAuditoriumWithCinemaName(appCtx))
	}

	{
		auditoriums := v1.Group("/auditoriums")
		//POST /v1/auditoriums
		auditoriums.POST("",
			middleware.RequireAuth(appCtx, userCaching),
			middleware.CheckRole(appCtx, "admin", "cinema_owner"),
			ginauditorium.CreateAuditorium(appCtx))
		//GET /v1/auditoriums/:id
		auditoriums.GET("/:id", ginauditorium.GetAuditoriumWithID(appCtx))
	}

	{
		companies := v1.Group("/companies")
		//GET /v1/companies
		companies.GET("", gincompany.ListCompany(appCtx))
		//GET /v1/companies/:id
		companies.GET("/:id", gincompany.GetCompanyWithID(appCtx))
		//POST /v1/companies
		companies.POST("", gincompany.CreateCompany(appCtx))
	}

	{
		movies := v1.Group("/movies")
		//GET /v1/movies
		movies.GET("", ginmovie.ListMovie(appCtx))
		//GET /v1/movies/:imdb_id
		movies.GET("/:imdb_id", ginmovie.GetMovieWithID(appCtx))
		//POST /v1/movies
		movies.POST("",
			middleware.RequireAuth(appCtx, userCaching),
			middleware.CheckRole(appCtx, "admin"),
			ginmovie.CreateMovie(appCtx))
	}

	{
		shows := v1.Group("/shows")
		//GET /v1/shows
		shows.GET("", ginshow.ListShow(appCtx))
		//GET /v1/shows/:id
		shows.GET("/:id", ginshow.GetShowWithID(appCtx))
		//POST /v1/shows
		shows.POST("", middleware.RequireAuth(appCtx, userCaching), ginshow.CreateShow(appCtx))
	}

	{
		tickets := v1.Group("/tickets")
		//GET /v1/tickets
		tickets.GET("", ginticket.ListTickets(appCtx))
		//GET /v1/tickets/user
		tickets.GET("/user", middleware.RequireAuth(appCtx, userCaching), ginticket.GetTicketsByUser(appCtx))
		//UPDATE /v1/tickets/
		tickets.PUT("", middleware.RequireAuth(appCtx, userCaching), ginticket.UpdateTicket(appCtx))
	}

	{
		genres := v1.Group("/genres")
		//GET /v1/genres
		genres.GET("", gingenre.ListGenres(appCtx))
		//GET /v1/genres/:id/movies
		genres.GET("/:id/movies", ginmoviesgenres.ListMoviesByGenres(appCtx))
	}

	{
		users := v1
		//GET /v1/profile
		users.GET("/profile", middleware.RequireAuth(appCtx, userCaching), ginuser.GetProfile(appCtx))
		//PUT /v1/profile
		users.PUT("/profile", middleware.RequireAuth(appCtx, userCaching), ginuser.UpdateUser(appCtx))
		//POST /v1/register
		users.POST("/register", ginuser.Register(appCtx))
		//POST /v1/login
		users.POST("/login", ginuser.Login(appCtx))
	}
	if err := r.Run(); err != nil {
		log.Fatalln(err)
	}
}
