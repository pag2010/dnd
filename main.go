package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/web"
	"github.com/jmoiron/sqlx"
	"github.com/satori/go.uuid"
)

const conf = "config.txt"
const sqlstr = "Install.sqlx"
const maxCount = 6

type httpError struct { //переопределить метод marshal
	Code int    `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

type httpError2 struct { //переопределить метод marshal
	Code int    `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}

/*func (h *httpError) MarshalJSON() ([]byte, error) {
	if h.Code == 0 {
		return nil, nil
	}
	return json.Marshal(httpError2{Code: h.Code, Text: h.Text})
}*/

type Context struct {
	User         *sUser          `json:"-"`
	Player       *sConnectPlayer `json:"-"`
	Hero         *sHero          `json:"Hero,omitempty"`
	OtherPlayers []*sPlayer      `json:"OtherPlayers,omitempty"`
	Err          *httpError      `json:"error,omitempty"`
	Response     string          `json:"response,omitempty"`
	Data         string          `json:"data,omitempty"`
	Game         string          `json:"game,omitempty"`
	Role         int             `json:"role,omitempty"`
	Session      string          `json:"session,omitempty"`
	Manual       *sManual        `json:"manual,omitempty"`
	HeroList     []HeroToShow    `json:"heroes,omitempty"`
	GameSessions map[string]int  `json:"games,omitempty"`
}

type sUser struct {
	ID      int    `db:"id"` //nullint и nullstring в sql
	Login   string `db:"login"`
	Hash    string `db:"hash"`
	Session string `db:"session"`
	RoleId  int    `db:"roleId"`
	Game    string `db:"game"`
}

type pAuth struct {
	Login string `json:"login"`
	Hash  string `json:"hash"`
}

type pReg struct {
	Login string `json:"login"`
	Hash  string `json:"hash"`
	Token string `json:"token"`
}

type sConfig struct {
	DBConnect  string `json:"DataBase"`
	AdminToken string `json:"Token"`
}

type sPlayerInfo struct {
	Login   string `json:"login"`
	Session string `json:"session"`
}

type sConnectPlayer struct {
	PlayerInfo sPlayerInfo `json:"auth"`
	Game       string      `json:"game"`
	Hero       int         `json:"hero"`
	PlayerId   int         `json:"playerId"`
}

type sPlayer struct {
	Login   string `json:"login"`
	Id      int    `json:"-"`
	Hero    *sHero `json:"hero"`
	Session string `json:"-"`
	Role    int    `json:"-"`
}

type sGame struct {
	//Player []sPlayer //сделать map[string]*sPlayer
	Player map[string]*sPlayer
	//Count  int //убрать
	sync.RWMutex
}

type sHeroDB struct {
	Id                   int    `db:"Id" json:"id"`
	Name                 string `db:"Name" json:"name"`
	Prehistory           string `db:"Prehistory" json:"prehistory"`
	Exp                  int    `db:"Exp" json:"exp"`
	Speed                int    `db:"Speed" json:"speed"`
	HP                   int    `db:"HP" json:"hp"`
	HPmax                int    `db:"HPmax" json:"hpmax"`
	HitBonesMax          int    `db:"HitBonesMax" json:""`
	HitBones             int    `db:"HitBones" json:""`
	Strength             int    `db:"Strength" json:""`
	Perception           int    `db:"Perception" json:""`
	Endurance            int    `db:"Endurance" json:""`
	Charisma             int    `db:"Charisma" json:""`
	Intelligence         int    `db:"Intelligence" json:""`
	Agility              int    `db:"Agility" json:""`
	MasterBonus          int    `db:"MasterBonus" json:""`
	DeathSavingThrowGood int    `db:"DeathSavingThrowGood" json:""`
	DeathSavingThrowBad  int    `db:"DeathSavingThrowBad" json:""`
	TemporaryHP          int    `db:"TemporaryHP" json:""`
	AC                   int    `db:"AC" json:""`
	Initiative           int    `db:"Initiative" json:""`
	PassiveAttention     bool   `db:"PassiveAttention" json:""`
	Inspiration          bool   `db:"Inspiration" json:""`
	Ammo                 int    `db:"Ammo" json:""`
	Languages            string `db:"Languages" json:""`
	SavingThrowS         bool   `db:"SavingThrowS" json:""`
	SavingThrowP         bool   `db:"SavingThrowP" json:""`
	SavingThrowE         bool   `db:"SavingThrowE" json:""`
	SavingThrowC         bool   `db:"SavingThrowC" json:""`
	SavingThrowI         bool   `db:"SavingThrowI" json:""`
	SavingThrowA         bool   `db:"SavingThrowA" json:""`
	Athletics            bool   `db:"Athletics" json:""`
	Acrobatics           bool   `db:"Acrobatics" json:""`
	Juggle               bool   `db:"Juggle" json:""`
	Stealth              bool   `db:"Stealth" json:""`
	Magic                bool   `db:"Magic" json:""`
	History              bool   `db:"History" json:""`
	Analysis             bool   `db:"Analysis" json:""`
	Nature               bool   `db:"Nature" json:""`
	Religion             bool   `db:"Religion" json:""`
	AnimalCare           bool   `db:"AnimalCare" json:""`
	Insight              bool   `db:"Insight" json:""`
	Medicine             bool   `db:"Medicine" json:""`
	Attention            bool   `db:"Attention" json:""`
	Survival             bool   `db:"Survival" json:""`
	Deception            bool   `db:"Deception" json:""`
	Intimidation         bool   `db:"Intimidation" json:""`
	Performance          bool   `db:"Performance" json:""`
	Conviction           bool   `db:"Conviction" json:""`
	WeaponFirstId        int    `db:"WeaponFirstId"`
	WeaponSecondId       int    `db:"WeaponSecondId"`
	ArmorId              int    `db:"ArmorId"`
	ShieldId             int    `db:"ShieldId"`
}

type sWeaponDBfull struct {
	Id         int    `db:"Id" json:"id"`
	Name       string `db:"Name" json:"name"`
	Damage     string `db:"Damage" json:"damage"`
	DmgType    int    `db:"DmgType" json:"dmgtype"`
	WeaponType int    `db:"Type" json:"type"`
	Cost       int    `db:"Cost" json:"cost"`
	Weight     int    `db:"Weight" json:"weight"`
}

type sWeaponDB struct {
	Id    int `db:"Id" json:"id"`
	Count int `db:"CountW" json:"count"`
	/*Name       string `db:"Name" json:"name"`
	Damage     string `db:"Damage" json:"damage"`
	DmgType    int    `db:"DmgType" json:"dmgtype"`
	WeaponType int    `db:"Type" json:"type"`
	Cost       int    `db:"Cost" json:"cost"`
	Weight     int    `db:"Weight" json:"weight"`*/
}

type sHero struct { //Главная структура героя. Содержит версию БД и массив оружия
	HeroDB  *sHeroDB    `json:"heroInfo"`
	Weapons []sWeaponDB `json:"weapons,omitemty"`
	//Weapons []int `json:"weapons,omitemty"`
}

type HeroToShow struct { //Нужен только для показа списка героев. Содержит только id и имя
	Id   int    `db:"id"`
	Name string `db:"name"`
}

type sGameMap struct {
	m map[string]sGame
	//m map[string]sGame
	sync.RWMutex
}

type jsHero struct {
	Hero *sHero `json:"Hero"`
}

type sManual struct {
	Roles       []sRoles        `json:"roles,omitemty"`
	Weapons     []sWeaponDBfull `json:"weapons,omitemty"`
	DmgTypes    []sDmgType      `json:"dmgtype,omitemty"`
	WeaponTypes []sWeaponType   `json:"weapontypes,omitemty"`
	Classes     []sClass        `json:"classes,omitemty"`
	Armors      []sArmor        `json:"armor,omitemty"`
	ArmorTypes  []sArmorType    `json:"armortype,omitemty"`
}

type sDmgType struct {
	Id   int    `db:"Id" json:"id"`
	Name string `db:"Name" json:"name"`
}

type sWeaponType struct {
	Id   int    `db:"Id" json:"id"`
	Name string `db:"Name" json:"name"`
}

type sClass struct {
	Id      int    `db:"Id" json:"id"`
	Name    string `db:"Name" json:"name"`
	About   string `db:"About" json:"about"`
	BoneHit string `db:"BoneHit" json:"bonehit"`
}

type sRoles struct {
	Id    int    `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	About string `db:"about" json:"about"`
}

type sArmor struct {
	Id     int    `db:"Id" json:"id"`
	Name   string `db:"Name" json:"name"`
	AC     int    `db:"AC" json:"ac"`
	Type   int    `db:"Type" json:"type"`
	Cost   int    `db:"Cost" json:"cost"`
	Weight int    `db:"Weight" json:"weight"`
}

type sArmorType struct {
	Id   int    `db:"Id" json:"id"`
	Name string `db:"Name" json:"name"`
}

var config sConfig
var Conn *sqlx.DB

var GameMap sGameMap

//var GameMap map[string]sGame
var GameSessions map[string]int

var DBConnStr string
var TokenStr string
var Ini bool

func init() {
	flag.BoolVar(&Ini, "i", false, "If you want to install do it")
	flag.StringVar(&DBConnStr, "db", "", "Here you should place database connection string. For example UserName:Password@tcp(localhost:3306)/DataBaseName")
	flag.StringVar(&TokenStr, "token", "lol", "Here you should place admin token")
}

func main() {
	var err error
	flag.Parse()
	if Ini {
		err = InstallConfig()
		if err != nil {
			log.Fatal(err)
		}
		err = InstallDB()
		if err != nil {
			log.Fatal(err)
		}
	}
	err = LoadConfig()
	if err != nil {
		log.Printf(err.Error())
		return
	}
	GameMap.m = make(map[string]sGame)
	GameSessions = make(map[string]int)
	router := web.New(Context{}).Middleware(web.LoggerMiddleware).Middleware((*Context).ErrorHandler)
	//router := web.New(Context{}).Middleware((*Context).Logger).Middleware((*Context).ErrorHandler)
	router.Subrouter(Context{}, "/").Post("/ping", (*Context).Ping)
	router.Subrouter(Context{}, "/").Post("/reg", (*Context).Reg)
	router.Subrouter(Context{}, "/").Post("/auth", (*Context).Auth)
	router.Subrouter(Context{}, "/").Get("/manual", (*Context).GetManual)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParsePatch).Patch("/sw", (*Context).SaveWeapons)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParsePost).Middleware((*Context).CheckUserSession).Post("/newGame", (*Context).NewGame).Delete("/newGame", (*Context).DestroyGame)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParsePost).Middleware((*Context).CheckUserSession).Middleware((*Context).Reconnect).Middleware((*Context).LoadHero).Post("/connect", (*Context).Connect)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParsePost).Middleware((*Context).CheckUserSession).Delete("/connect", (*Context).Disconnect)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetls).Middleware((*Context).CheckUserSession).Get("/heroList", (*Context).GetHeroes)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetls).Middleware((*Context).CheckUserSession).Get("/games", (*Context).GetAvaliableGames)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetls).Middleware((*Context).CheckUserSession).Middleware((*Context).ParsePatch).Post("/newHero", (*Context).NewHero)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetgls).Middleware((*Context).CheckPlayerSession).Get("/:game/Other", (*Context).GetOtherPlayers)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetgls).Middleware((*Context).CheckPlayerSession).Get("/:game/Hero", (*Context).GetHero)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetgls).Middleware((*Context).CheckPlayerSession).Middleware((*Context).ParsePatch).Patch("/:game/Hero", (*Context).UpdateHero)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetgls).Middleware((*Context).CheckPlayerSession).Patch("/:game/SaveHero", (*Context).SaveHero)
	router.Subrouter(Context{}, "/").Middleware((*Context).ParseGetgls).Middleware((*Context).CheckPlayerSession).Patch("/:game/SaveGame", (*Context).SaveGame)

	fmt.Println("Запускаемся. Слушаем порт 8080")
	http.Handle("/", router)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Printf(err.Error())
		return
	}

}

func LoadConfig() error {
	file, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	Conn, err = sqlx.Connect("mysql", config.DBConnect)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	return nil
}

func (c *Context) NewGame(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.User.RoleId != 1 {
		c.SetError(403, "Новую игру может начать только GameMaster")
		return
	}

	if _, ok := GameMap.m[c.User.Game]; ok {
		c.Game = c.User.Game
		return
	}
	session, err := uuid.NewV4()
	if err != nil {
		c.SetError(500, "Не удалось создать игру")
		log.Printf(err.Error())
		return
	}

	_, err = Conn.Exec("update users set game=? where id=?", session, c.User.ID)
	if err != nil {
		c.SetError(500, "Невозможно подключиться к игровой сессии")
		return
	}

	var game sGame
	//game.Player = make([]sPlayer, maxCount)
	game.Player = make(map[string]*sPlayer)
	game.Player[c.User.Login] = new(sPlayer)
	game.Player[c.User.Login].Id = c.User.ID
	game.Player[c.User.Login].Login = c.User.Login
	game.Player[c.User.Login].Hero = nil
	game.Player[c.User.Login].Session = c.User.Session
	game.Player[c.User.Login].Role = c.User.RoleId

	GameMap.Lock()
	GameMap.m[session.String()] = game
	GameMap.Unlock()
	//GameSessions = append(GameSessions, session.String())
	GameSessions[session.String()] = 1

	c.Game = session.String()
	return
}

func (c *Context) ParsePost(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newPlayer sConnectPlayer
	err := buf.Decode(&newPlayer)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		//log.Printf(err.Error())
		return
	}
	c.Player = &newPlayer
	next(iWrt, iReq)
}

func (c *Context) CheckUserSession(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	/*buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newPlayer sConnectPlayer
	err := buf.Decode(&newPlayer)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		//log.Printf(err.Error())
		return
	}*/

	user := []sUser{}
	err := Conn.Select(&user, "select * from users where login=? and session=?", c.Player.PlayerInfo.Login, c.Player.PlayerInfo.Session)
	if err != nil {
		c.SetError(500, "Ошибка БД")
	}
	if len(user) != 1 {
		c.SetError(401, "Неверный логин или ключ сессии. Нужна повторная авторизация")
		return
	}
	c.User = &user[0]
	//c.Player = &newPlayer
	next(iWrt, iReq)
	return
}

func (c *Context) Reconnect(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	gameSession := c.User.Game
	if _, ok := GameMap.m[gameSession]; ok {
		fmt.Println("Беру данные из map")
		c.Hero = GameMap.m[gameSession].Player[c.User.Login].Hero
		c.Game = gameSession
		return
	} else {
		if c.Player.Game == "" && c.Player.Hero == 0 {
			c.SetError(404, "Не удалось подключиться к последней игре")
			return
		}
		fmt.Println("Не удалось переподключиться")
		next(iWrt, iReq)
		return
	}
}

func (c *Context) Connect(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.User.RoleId != 2 {
		c.SetError(403, "Подключиться к сессии может только player")
		return
	}
	gameSession := c.Player.Game

	if c.Hero == nil {
		c.SetError(404, "При подключении к игре не был обнаружен герой")
		return
	}

	if len(GameMap.m[gameSession].Player) == maxCount {
		c.SetError(403, "Подключиться к сессии не удалось. Сессия заполнена")
		return
	}

	GameMap.Lock()
	defer GameMap.Unlock()
	var Player sPlayer
	Player.Login = c.User.Login
	Player.Id = c.User.ID
	Player.Session = c.User.Session
	Player.Hero = c.Hero
	Player.Role = c.User.RoleId
	if game, ok := GameMap.m[gameSession]; ok {
		_, err := Conn.Exec("update users set game=? where id=?", gameSession, c.User.ID)
		if err != nil {
			c.SetError(500, "Невозможно подключиться к игровой сессии")
			return
		}

		game.Player[Player.Login] = &Player
		GameMap.m[gameSession] = game
		GameSessions[gameSession]++
		if len(GameMap.m[gameSession].Player) == maxCount {
			delete(GameSessions, gameSession) // проще пробежать по всей map
		}
		return
	} else {
		c.SetError(404, "Игровая сессия не найдена")
	}
	return
}

func (c *Context) Disconnect(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.User.RoleId != 2 {
		c.SetError(403, "Отключиться может только игрок")
		return
	}
	GameMap.Lock()
	defer GameMap.Unlock()
	if _, ok := GameMap.m[c.User.Game]; ok {
		_, err := Conn.Exec("Update users set game = '' where id=?", c.User.ID)
		if err != nil {
			c.SetError(500, "Ошибка при удалении игры")
			return
		}
		GameSessions[c.User.Game]--
		delete(GameMap.m[c.User.Game].Player, c.User.Login)
		c.Response = "true"
	} else {
		c.SetError(404, "Игра не обнаружена")
		_, err := Conn.Exec("Update users set game = '' where id =?", c.User.ID)
		if err != nil {
			c.SetError(500, "Ошибка при удалении игры")
			return
		}
	}
}

func (c *Context) GetHeroes(iWrt web.ResponseWriter, iReq *web.Request) { //TODO исправить make
	//fmt.Println(c.User.Login)
	c.HeroList = make([]HeroToShow, 0)
	c.LoadHeroList(c.User.ID)
	//fmt.Println(c.HeroList[0])
	return
}

func (c *Context) LoadHeroList(idUser int) (err error) {
	//h = nil
	//k := new([]HeroToShow)
	heroes := []HeroToShow{}
	err = Conn.Select(&heroes, "select heroes.id, name from herotouser inner join heroes on idhero = heroes.id where iduser = ?", idUser)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	c.HeroList = heroes
	return nil
}

func (c *Context) LoadHero(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	hero := []sHeroDB{}
	//err := Conn.Select(&hero, "select * from Heroes where id=?", c.Player.Hero)
	err := Conn.Select(&hero, "select heroes.* from heroes inner join herotouser on heroes.id=herotouser.IdHero where herotouser.IdUser = ? and herotouser.IdHero=?", c.User.ID, c.Player.Hero)
	if err != nil {
		c.SetError(500, "Невозможно загрузить героя из БД")
		return
	}
	if len(hero) != 1 {
		c.SetError(404, "Не удалось найти героя")
		return
	}
	c.Hero = new(sHero)
	c.Hero.HeroDB = &hero[0]
	c.Hero.LoadWeapons()
	next(iWrt, iReq)
}

func (h *sHero) LoadWeapons() error {
	//h.Weapons = make([]sWeaponDB, 2)
	weapons := []sWeaponDB{}
	//weapons := []int{}
	//err := Conn.Select(&weapons, "SELECT weapons.Id, weapons.Name, weapons.Damage, dmgtype.name as 'DmgType', weapontype.Name as 'WeaponType', weapons.Cost, weapons.Weight from weapons inner join dmgtype on weapons.dmgtype = dmgtype.id inner join weapontype on weapons.Type= weapontype.id where weapons.Id = ? and weapons.Id = ?", h.HeroDB.WeaponFirstId, h.HeroDB.WeaponSecondId)
	/*err := Conn.Select(&weapons, "SELECT id from weapons where weapons.Id = ? or weapons.Id = ?", h.HeroDB.WeaponFirstId, h.HeroDB.WeaponSecondId)
	if err != nil {
		return err
	}*/
	//h.Weapons = weapons
	err := Conn.Select(&weapons, "SELECT WeaponId as 'Id', CountW from HeroToWeapons where HeroId = ?", h.HeroDB.Id) //kek
	fmt.Println(weapons[0].Id)
	if err != nil {
		return err
	}
	h.Weapons = weapons
	//h.Weapons = append(h.Weapons, weapons...)
	return nil
}

func (c *Context) SaveWeapons(iWrt web.ResponseWriter, iReq *web.Request) {
	weapons := []sWeaponDB{}
	//c.Hero.Weapons = append(c.Hero.Weapons, sWeaponDB{1, 1})
	err := Conn.Select(&weapons, "Select weaponid as 'Id', CountW from herotoweapons where heroid=?", c.Hero.HeroDB.Id)
	if err != nil {
		fmt.Println(err.Error())
		c.SetError(500, "Невозможно получить список оружия героя из БД")
		return
	}
	oldw := make(map[int]int)
	neww := make(map[int]int)
	delm := []int{}
	//neww := []sWeaponDB{}
	updw := []sWeaponDB{}
	insw := []sWeaponDB{}
	var w sWeaponDB
	if c.Hero == nil {
		c.SetError(406, "Оружие героя nil")
		return
	}
	for _, i := range weapons {
		oldw[i.Id] = i.Count
	}
	for _, i := range c.Hero.Weapons {
		neww[i.Id] = i.Count
	}
	for id, count := range neww {
		if c, ok := oldw[id]; ok {
			if c != count {
				w.Id = id
				w.Count = count
				updw = append(updw, w)
			}
		} else {
			w.Id = id
			w.Count = count
			insw = append(insw, w)
		}
	}

	for id, _ := range oldw {
		if _, ok := neww[id]; !ok {
			delm = append(delm, id)
		}
	}

	fmt.Printf("Update %+v\n", updw)
	fmt.Printf("Insert %+v\n", insw)
	fmt.Printf("Delete %+v\n", delm)

	return
}

func (c *Context) DestroyGame(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.User.RoleId != 1 {
		c.SetError(403, "Закончить игру может только master")
		return
	}
	GameMap.Lock()
	defer GameMap.Unlock()
	if _, ok := GameMap.m[c.User.Game]; ok {
		_, err := Conn.Exec("Update users set game = '' where id in (Select id from (Select id from users where game = ?)as a)", c.User.Game)
		if err != nil {
			c.SetError(500, "Ошибка при удалении игры")
			return
		}
		delete(GameMap.m, c.User.Game)
		c.Response = "true"
	} else {
		c.SetError(404, "Игра не обнаружена")
		_, err := Conn.Exec("Update users set game = '' where id =?", c.User.ID)
		if err != nil {
			c.SetError(500, "Ошибка при удалении игры")
			return
		}
	}
}

func (c *Context) SaveHero(iWrt web.ResponseWriter, iReq *web.Request) {
	/*if c.User.RoleId != 2 {
		c.SetError(403, "Сохранить героя может только игрок")
		return
	}*/
	if game, ok := GameMap.m[c.User.Game]; ok {
		//if game.Player[c.Player.PlayerId].Id == c.User.ID {
		//h := game.Player[c.Player.PlayerId].Hero.HeroDB
		if p, ok := game.Player[c.User.Login]; ok {
			if p.Hero == nil {
				c.SetError(404, "Герой не найден")
				return
			}
			if p.Hero == nil {
				c.SetError(404, "Герой не найден")
				return
			}
			if p.Role != 2 {
				c.SetError(403, "Сохранить героя может только игрок")
				return
			}
			h := p.Hero.HeroDB
			_, err := Conn.Exec("update heroes set Exp=?, Speed=?, HP=?, HPmax=?, HitBonesMax=?, HitBones=?, Strength=?, Perception=?, Endurance=?, Charisma=?, Intelligence=?, Agility=?, MasterBonus=?, DeathSavingThrowGood=?, DeathSavingThrowBad=?, TemporaryHP=?, AC=?, Initiative=?, PassiveAttention=?, Inspiration=?, Ammo=?, Languages=?, SavingThrowS=?, SavingThrowP=?, SavingThrowE=?, SavingThrowC=?, SavingThrowI=?, SavingThrowA=?, Athletics=?, Acrobatics=?, Juggle=?, Stealth=?, Magic=?, History=?, Analysis=?, Nature=?, Religion=?, AnimalCare=?, Insight=?, Medicine=?, Attention=?, Survival=?, Deception=?, Intimidation=?, Performance=?, Conviction=?, WeaponFirstId=?, WeaponSecondId=?, ArmorId=?, ShieldId=? where Id=?", h.Exp, h.Speed, h.HP, h.HPmax, h.HitBonesMax, h.HitBones, h.Strength, h.Perception, h.Endurance, h.Charisma, h.Intelligence, h.Agility, h.MasterBonus, h.DeathSavingThrowGood, h.DeathSavingThrowBad, h.TemporaryHP, h.AC, h.Initiative, h.PassiveAttention, h.Inspiration, h.Ammo, h.Languages, h.SavingThrowS, h.SavingThrowP, h.SavingThrowE, h.SavingThrowC, h.SavingThrowI, h.SavingThrowA, h.Athletics, h.Acrobatics, h.Juggle, h.Stealth, h.Magic, h.History, h.Analysis, h.Nature, h.Religion, h.AnimalCare, h.Insight, h.Medicine, h.Attention, h.Survival, h.Deception, h.Intimidation, h.Performance, h.Conviction, h.WeaponFirstId, h.WeaponSecondId, h.ArmorId, h.ShieldId, h.Id)
			if err != nil {
				c.SetError(500, "Невозможно сохранить героя")
				fmt.Println(err.Error())
				return
			}
			c.Response = "true"
			return
		} else {
			c.SetError(404, "Игрок не найден в игровой сессии")
			return
		}
	} else {
		c.SetError(404, "Игровая сессия не найдена")
		return
	}

}

func (c *Context) SaveGame(iWrt web.ResponseWriter, iReq *web.Request) {
	/*if c.User.RoleId != 1 {
		c.SetError(403, "Сохранить игру может только мастер")
		return
	}*/
	if game, ok := GameMap.m[c.User.Game]; ok {
		if game.Player[c.User.Login].Role != 1 {
			c.SetError(403, "Сохранить игру может только мастер")
			return
		}
		var e error
		tx, err := Conn.Begin()
		if err != nil {
			c.SetError(500, "Невозможно начать транзакцию с БД")
			return
		}
		for _, i := range game.Player {
			if i.Hero != nil {
				h := i.Hero.HeroDB
				_, err := tx.Exec("update heroes set Exp=?, Speed=?, HP=?, HPmax=?, HitBonesMax=?, HitBones=?, Strength=?, Perception=?, Endurance=?, Charisma=?, Intelligence=?, Agility=?, MasterBonus=?, DeathSavingThrowGood=?, DeathSavingThrowBad=?, TemporaryHP=?, AC=?, Initiative=?, PassiveAttention=?, Inspiration=?, Ammo=?, Languages=?, SavingThrowS=?, SavingThrowP=?, SavingThrowE=?, SavingThrowC=?, SavingThrowI=?, SavingThrowA=?, Athletics=?, Acrobatics=?, Juggle=?, Stealth=?, Magic=?, History=?, Analysis=?, Nature=?, Religion=?, AnimalCare=?, Insight=?, Medicine=?, Attention=?, Survival=?, Deception=?, Intimidation=?, Performance=?, Conviction=?, WeaponFirstId=?, WeaponSecondId=?, ArmorId=?, ShieldId=? where Id=?", h.Exp, h.Speed, h.HP, h.HPmax, h.HitBonesMax, h.HitBones, h.Strength, h.Perception, h.Endurance, h.Charisma, h.Intelligence, h.Agility, h.MasterBonus, h.DeathSavingThrowGood, h.DeathSavingThrowBad, h.TemporaryHP, h.AC, h.Initiative, h.PassiveAttention, h.Inspiration, h.Ammo, h.Languages, h.SavingThrowS, h.SavingThrowP, h.SavingThrowE, h.SavingThrowC, h.SavingThrowI, h.SavingThrowA, h.Athletics, h.Acrobatics, h.Juggle, h.Stealth, h.Magic, h.History, h.Analysis, h.Nature, h.Religion, h.AnimalCare, h.Insight, h.Medicine, h.Attention, h.Survival, h.Deception, h.Intimidation, h.Performance, h.Conviction, h.WeaponFirstId, h.WeaponSecondId, h.ArmorId, h.ShieldId, h.Id)
				if err != nil {
					e = err
				}
			}
		}
		if e != nil {
			c.SetError(500, "Невозможно сохранить игру")
			tx.Rollback()
			return
		}
		tx.Commit()
		c.Response = "true"
		return

	}

}

func (c *Context) Reg(iWrt web.ResponseWriter, iReq *web.Request) {
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newUser pReg
	err := buf.Decode(&newUser)

	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		log.Printf(err.Error())
		return
	}
	if config.AdminToken != newUser.Token {
		c.SetError(403, "Неверный хэш администратора")
		log.Println("Токены не совпали")
		return
	}

	fmt.Println("Началась бд")
	user := []sUser{}
	err = Conn.Select(&user, "select * from users where login=?", newUser.Login)

	if err != nil {
		c.SetError(500, "Ошибка базы данных")
		log.Printf(err.Error())
		return
	}

	if len(user) == 0 {
		_, err = Conn.Exec("insert into users (login, hash) values (?,?)", newUser.Login, newUser.Hash)
		if err != nil {
			log.Printf(err.Error())
			c.SetError(500, "Ошибка базы данных")
			return
		}
		c.Response = newUser.Login
		return

	} else {
		c.SetError(401, "Такой пользователь уже существует")
		return
	}
}

func (c *Context) Auth(iWrt web.ResponseWriter, iReq *web.Request) {
	var user sUser

	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var p pAuth

	err := buf.Decode(&p)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		log.Printf(err.Error())
		return
	}

	fmt.Println(p.Login)
	err = Conn.Get(&user, "select * from users where login=?", p.Login)
	if err != nil {
		c.SetError(401, "Неверный логин или пароль")
		log.Printf(err.Error())
		return
	}
	if user.Hash == p.Hash {

		session, err := uuid.NewV4()
		if err != nil {
			c.SetError(500, "Не удалось создать сессию")
			log.Printf(err.Error())
			return
		}
		user.Session = session.String()
		_, err = Conn.Exec("update users set session =? where id=?", user.Session, user.ID)
		if err != nil {
			c.SetError(500, "Не удалось создать сессию")
			log.Printf(err.Error())
			return
		}
		if game, ok := GameMap.m[user.Game]; ok { //обновление сессии в игровой карте, если эта игра и игрок там присутствуют
			if pl, ok := game.Player[user.Login]; ok {
				pl.Session = session.String()
			}
		}
		//c.Response = user.Session
		c.Session = user.Session
		c.Role = user.RoleId
		if err != nil {
			c.SetError(500, "Невозможно преобразовать ответ в json")
			log.Printf(err.Error())
			return
		}
	} else {
		c.SetError(403, "Неверный логин или пароль")
		return
	}
	return
}

func (c *Context) ErrorHandler(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	next(iWrt, iReq)
	if c.Err != nil {
		iWrt.WriteHeader(c.Err.Code)
		lData, err := json.Marshal(c.Err)
		if err != nil {
			iWrt.WriteHeader(500)
			fmt.Fprintln(iWrt, "")
		}
		fmt.Fprintln(iWrt, string(lData))
	} else {
		lData, err := json.Marshal(c)
		if err != nil {
			iWrt.WriteHeader(500)
			fmt.Fprintln(iWrt, "")
		}
		fmt.Fprintln(iWrt, string(lData))
	}
}

func (c *Context) SetError(code int, text string, args ...interface{}) {
	if text != "" {
		c.Err = new(httpError)
		c.Err.Code = code
		if len(args) == 0 {
			c.Err.Text = text
		} else {
			c.Err.Text = fmt.Sprintf(text, args...)
		}
	}
}

func (c *Context) Logger(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	t := time.Now()
	next(iWrt, iReq)
	if c.Err != nil {
		fmt.Printf("[ %s ] %d %s\n", time.Since(t), 200, iReq.URL)
		return
	}
	fmt.Printf("[ %s ] %d %s\n", time.Since(t), 200, iReq.URL)
}

func (c *Context) GetOtherPlayers(iWrt web.ResponseWriter, iReq *web.Request) {
	if GameMap.m[c.User.Game].Player[c.User.Login].Role != 1 {
		c.SetError(403, "Доступ только у ГМ")
		return
	}
	for _, i := range GameMap.m[c.User.Game].Player {
		c.OtherPlayers = append(c.OtherPlayers, i)
	}
}

func (c *Context) ParseGetgls(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) { //:game login session
	iReq.ParseForm()
	game := iReq.PathParams["game"]
	login := iReq.Form["login"]
	session := iReq.Form["session"]
	c.User = new(sUser)
	c.User.Login = strings.Join(login, "")
	c.User.Session = strings.Join(session, "")
	c.User.Game = game
	next(iWrt, iReq)
}

func (c *Context) ParseGetls(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) { //login session
	iReq.ParseForm()
	login := iReq.Form["login"]
	session := iReq.Form["session"]
	c.Player = new(sConnectPlayer)
	c.Player.PlayerInfo.Login = strings.Join(login, "")
	c.Player.PlayerInfo.Session = strings.Join(session, "")
	next(iWrt, iReq)
}

func (c *Context) ParsePatch(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()
	var Herojson jsHero
	err := buf.Decode(&Herojson)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		return
	}
	c.Hero = Herojson.Hero
	next(iWrt, iReq)
}

func (c *Context) CheckPlayerSession(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	if game, ok := GameMap.m[c.User.Game]; ok {
		if pl, ok := game.Player[c.User.Login]; ok {
			if c.User.Session == pl.Session {
				//c.Response = "Сессии одинаковые"
				next(iWrt, iReq)
				return
			} else {
				c.SetError(403, "Сессии разные")
				return
			}
		} else {
			c.SetError(404, "Игрок не найден")
			return
		}
	} else {
		c.SetError(404, "Игра не найдена")
		return
	}
}

func (c *Context) GetHero(iWrt web.ResponseWriter, iReq *web.Request) {
	c.Hero = GameMap.m[c.User.Game].Player[c.User.Login].Hero
}

func (c *Context) UpdateHero(iWrt web.ResponseWriter, iReq *web.Request) {
	GameMap.m[c.User.Game].Player[c.User.Login].Hero = c.Hero
	c.Hero = nil
	c.Response = "true"
}

func (c *Context) GetManual(iWrt web.ResponseWriter, iReq *web.Request) {
	c.Manual = new(sManual)
	err := Conn.Select(&c.Manual.Weapons, "SELECT * from weapons")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника ОРУЖИЕ")
		return
	}

	err = Conn.Select(&c.Manual.Roles, "SELECT * from roles")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника РОЛИ")
		return
	}

	err = Conn.Select(&c.Manual.DmgTypes, "SELECT * from dmgtype")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника ТИП УРОНА")
		return
	}

	err = Conn.Select(&c.Manual.WeaponTypes, "SELECT * from weapontype")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника ТИП ОРУЖИЯ")
		return
	}

	err = Conn.Select(&c.Manual.Classes, "SELECT * from classes")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника КЛАССЫ")
		return
	}

	err = Conn.Select(&c.Manual.ArmorTypes, "SELECT * from armortype")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника ТИП БРОНИ")
		return
	}

	err = Conn.Select(&c.Manual.Armors, "SELECT * from armors")
	if err != nil {
		c.SetError(500, "Ошибка загрузки справочника БРОНЯ")
		return
	}

}

func (c *Context) Ping(iWrt web.ResponseWriter, iReq *web.Request) {
	c.Response = "Alive"
}

func (c *Context) GetAvaliableGames(iWrt web.ResponseWriter, iReq *web.Request) {
	c.GameSessions = GameSessions
}

func (c *Context) NewHero(iWrt web.ResponseWriter, iReq *web.Request) {
	/*_, err := Conn.Exec("Insert into heroes (Name, Prehistory, Exp, Speed, HP, HPmax, HitBonesMax, HitBones, Strength, Perception, Endurance, Charisma, Intelligence, Agility, MasterBonus, DeathSavingThrowGood, DeathSavingThrowBad, TemporaryHP, AC, Initiative, PassiveAttention, Inspiration, Ammo, Languages, SavingThrowS, SavingThrowP, SavingThrowE, SavingThrowC, SavingThrowI, SavingThrowA, Athletics, Acrobatics, Juggle, Stealth, Magic, History, Analysis, Nature, Religion, AnimalCare, Insight, Medicine, Attention, Survival, Deception, Intimidation, Performance, Conviction, WeaponFirstId, WeaponSecondId, ArmorId, ShieldId) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", c.Hero.HeroDB.Name, c.Hero.HeroDB.Prehistory, c.Hero.HeroDB.Exp, c.Hero.HeroDB.Speed, c.Hero.HeroDB.HP, c.Hero.HeroDB.HPmax, c.Hero.HeroDB.HitBonesMax, c.Hero.HeroDB.HitBones, c.Hero.HeroDB.Strength, c.Hero.HeroDB.Perception, c.Hero.HeroDB.Endurance, c.Hero.HeroDB.Charisma, c.Hero.HeroDB.Intelligence, c.Hero.HeroDB.Agility, c.Hero.HeroDB.MasterBonus, c.Hero.HeroDB.DeathSavingThrowGood, c.Hero.HeroDB.DeathSavingThrowBad, c.Hero.HeroDB.TemporaryHP, c.Hero.HeroDB.AC, c.Hero.HeroDB.Initiative, c.Hero.HeroDB.PassiveAttention, c.Hero.HeroDB.Inspiration, c.Hero.HeroDB.Ammo, c.Hero.HeroDB.Languages, c.Hero.HeroDB.SavingThrowS, c.Hero.HeroDB.SavingThrowP, c.Hero.HeroDB.SavingThrowE, c.Hero.HeroDB.SavingThrowC, c.Hero.HeroDB.SavingThrowI, c.Hero.HeroDB.SavingThrowA, c.Hero.HeroDB.Athletics, c.Hero.HeroDB.Acrobatics, c.Hero.HeroDB.Juggle, c.Hero.HeroDB.Stealth, c.Hero.HeroDB.Magic, c.Hero.HeroDB.History, c.Hero.HeroDB.Analysis, c.Hero.HeroDB.Nature, c.Hero.HeroDB.Religion, c.Hero.HeroDB.AnimalCare, c.Hero.HeroDB.Insight, c.Hero.HeroDB.Medicine, c.Hero.HeroDB.Attention, c.Hero.HeroDB.Survival, c.Hero.HeroDB.Deception, c.Hero.HeroDB.Intimidation, c.Hero.HeroDB.Performance, c.Hero.HeroDB.Conviction, c.Hero.HeroDB.WeaponFirstId, c.Hero.HeroDB.WeaponSecondId, c.Hero.HeroDB.ArmorId, c.Hero.HeroDB.ShieldId)
	if err != nil {
		fmt.Println(err.Error())
		c.SetError(500, "Невозможно добавить героя в БД")
	}*/

	var e error
	tx, err := Conn.Begin()
	if err != nil {
		c.SetError(500, "Невозможно начать транзакцию с БД")
		return
	}
	result, err := Conn.Exec("Insert into heroes (Name, Prehistory, Exp, Speed, HP, HPmax, HitBonesMax, HitBones, Strength, Perception, Endurance, Charisma, Intelligence, Agility, MasterBonus, DeathSavingThrowGood, DeathSavingThrowBad, TemporaryHP, AC, Initiative, PassiveAttention, Inspiration, Ammo, Languages, SavingThrowS, SavingThrowP, SavingThrowE, SavingThrowC, SavingThrowI, SavingThrowA, Athletics, Acrobatics, Juggle, Stealth, Magic, History, Analysis, Nature, Religion, AnimalCare, Insight, Medicine, Attention, Survival, Deception, Intimidation, Performance, Conviction, WeaponFirstId, WeaponSecondId, ArmorId, ShieldId) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", c.Hero.HeroDB.Name, c.Hero.HeroDB.Prehistory, c.Hero.HeroDB.Exp, c.Hero.HeroDB.Speed, c.Hero.HeroDB.HP, c.Hero.HeroDB.HPmax, c.Hero.HeroDB.HitBonesMax, c.Hero.HeroDB.HitBones, c.Hero.HeroDB.Strength, c.Hero.HeroDB.Perception, c.Hero.HeroDB.Endurance, c.Hero.HeroDB.Charisma, c.Hero.HeroDB.Intelligence, c.Hero.HeroDB.Agility, c.Hero.HeroDB.MasterBonus, c.Hero.HeroDB.DeathSavingThrowGood, c.Hero.HeroDB.DeathSavingThrowBad, c.Hero.HeroDB.TemporaryHP, c.Hero.HeroDB.AC, c.Hero.HeroDB.Initiative, c.Hero.HeroDB.PassiveAttention, c.Hero.HeroDB.Inspiration, c.Hero.HeroDB.Ammo, c.Hero.HeroDB.Languages, c.Hero.HeroDB.SavingThrowS, c.Hero.HeroDB.SavingThrowP, c.Hero.HeroDB.SavingThrowE, c.Hero.HeroDB.SavingThrowC, c.Hero.HeroDB.SavingThrowI, c.Hero.HeroDB.SavingThrowA, c.Hero.HeroDB.Athletics, c.Hero.HeroDB.Acrobatics, c.Hero.HeroDB.Juggle, c.Hero.HeroDB.Stealth, c.Hero.HeroDB.Magic, c.Hero.HeroDB.History, c.Hero.HeroDB.Analysis, c.Hero.HeroDB.Nature, c.Hero.HeroDB.Religion, c.Hero.HeroDB.AnimalCare, c.Hero.HeroDB.Insight, c.Hero.HeroDB.Medicine, c.Hero.HeroDB.Attention, c.Hero.HeroDB.Survival, c.Hero.HeroDB.Deception, c.Hero.HeroDB.Intimidation, c.Hero.HeroDB.Performance, c.Hero.HeroDB.Conviction, c.Hero.HeroDB.WeaponFirstId, c.Hero.HeroDB.WeaponSecondId, c.Hero.HeroDB.ArmorId, c.Hero.HeroDB.ShieldId)
	if err != nil {
		fmt.Println(err.Error())
		e = err
		c.SetError(500, "Невозможно добавить героя в БД")
	}
	id, err := result.LastInsertId()
	if err != nil {
		e = err
		c.SetError(500, "Нежданчик")
	}
	_, err = Conn.Exec("Insert into herotouser (idhero, iduser) values (?, ?)", id, c.User.ID)
	if err != nil {
		fmt.Println(err.Error())
		e = err
		c.SetError(500, "Невозможно связать героя и пользователя")
	}
	if e != nil {
		c.SetError(500, "Невозможно добавить героя в БД")
		tx.Rollback()
		return
	}
	tx.Commit()
	c.Response = "true"
}

func InstallDB() error {
	b, err := ioutil.ReadFile(sqlstr)
	if err != nil {
		return err
	}
	var conn *sqlx.DB
	fmt.Println(DBConnStr)
	fmt.Println(TokenStr)
	conn, err = sqlx.Connect("mysql", DBConnStr)
	if err != nil {
		return err
	}
	_, err = conn.Exec(string(b))
	if err != nil {
		return err
	}
	return nil
}

func InstallConfig() error {
	var c sConfig
	c.AdminToken = TokenStr
	c.DBConnect = DBConnStr
	lData, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(conf, lData, os.ModePerm)
	return err
}
