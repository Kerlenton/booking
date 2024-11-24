package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var db Database

// Определяем интерфейс для базы данных
type Database interface {
	AutoMigrate(models ...interface{}) error
	Create(value interface{}) error
	Where(query interface{}, args ...interface{}) Database
	First(dest interface{}, conds ...interface{}) error
	Find(dest interface{}, conds ...interface{}) error
	Transaction(fc func(tx *gorm.DB) error) error
}

type GormDatabase struct {
	Conn *gorm.DB
}

func (g *GormDatabase) AutoMigrate(models ...interface{}) error {
	return g.Conn.AutoMigrate(models...)
}

func (g *GormDatabase) Create(value interface{}) error {
	return g.Conn.Create(value).Error
}

func (g *GormDatabase) Where(query interface{}, args ...interface{}) Database {
	tx := g.Conn.Where(query, args...)
	return &GormDatabase{Conn: tx}
}

func (g *GormDatabase) First(dest interface{}, conds ...interface{}) error {
	return g.Conn.First(dest, conds...).Error
}

func (g *GormDatabase) Find(dest interface{}, conds ...interface{}) error {
	return g.Conn.Find(dest, conds...).Error
}

func (g *GormDatabase) Transaction(fc func(tx *gorm.DB) error) error {
	return g.Conn.Transaction(fc)
}

type Booking struct {
	gorm.Model
	RoomName  string    `json:"room_name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	UserID    uint      `json:"user_id"`
}

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("failed to get DB from gorm: %v", err)
	}
	// Настраиваем пул соединений
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	database.AutoMigrate(&Booking{})
	db = &GormDatabase{Conn: database}
}

func validateToken(tokenStr string) (*jwt.Token, error) {
	claims := &Claims{}
	return jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("secret"), nil
	})
}

func createBooking(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tokenStr := strings.Split(authHeader, "Bearer ")[1]
	token, err := validateToken(tokenStr)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	claims := token.Claims.(*Claims)
	var booking Booking
	if err := c.ShouldBindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Привязываем бронирование к пользователю
	booking.UserID = claims.UserID

	// Используем транзакцию для проверки и создания бронирования
	err = db.Transaction(func(tx *gorm.DB) error {
		var existingBooking Booking
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("room_name = ? AND start_time < ? AND end_time > ?",
			booking.RoomName, booking.EndTime, booking.StartTime).First(&existingBooking).Error; err == nil {
			return fmt.Errorf("room is already booked for this time")
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(&booking).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if err.Error() == "room is already booked for this time" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			log.Printf("Transaction error: %v", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Booking created successfully"})
}

func getBookings(c *gin.Context) {
	var bookings []Booking
	if err := db.Find(&bookings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve bookings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookings": bookings})
}

func main() {
	initDB()
	r := gin.Default()

	// Добавление CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/book", createBooking)
	r.GET("/bookings", getBookings)
	r.Run(":8082")
}
