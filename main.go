package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/web"
	"github.com/jmoiron/sqlx"
	"github.com/satori/go.uuid"
)

const conf = "config.txt"
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
	User     *sUser          `json:"-"`
	Player   *sConnectPlayer `json:"-"`
	Hero     *sHero          `json:"Hero,omitempty"`
	Err      *httpError      `json:"error,omitempty"`
	Response string          `json:"response,omitempty"`
	Data     string          `json:"data,omitempty"`
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
}

type sPlayer struct {
	//PlayerInfo sPlayerInfo
	Login string
	Id    int
	Hero  *sHero //Заменить на sHero
}

type sGame struct {
	Player []sPlayer
	Count  int
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

type sWeaponDB struct {
	Id         int    `db:"Id" json:""`
	Name       string `db:"Name" json:""`
	Damage     string `db:"Damage" json:""`
	DmgType    string `db:"DmgType" json:""`
	WeaponType string `db:"WeaponType" json:""`
	Cost       int    `db:"Cost" json:""`
	Weight     int    `db:"Weight" json:""`
}

type sHero struct { //Главная структура героя. Содержит версию БД и массив оружия
	HeroDB  *sHeroDB    `json:"hero"`
	Weapons []sWeaponDB `json:"weapons,omitemty"`
}

type HeroToShow struct { //Нужен только для показа списка героев. Содержит только id и имя
	Id   int    `db:"id"`
	Name string `db:"name"`
}

var config sConfig
var Conn *sqlx.DB
var GameMap map[string]sGame
var GameSessions map[string]int

func main() {
	err := LoadConfig()
	if err != nil {
		log.Printf(err.Error())
		return
	}
	GameMap = make(map[string]sGame)
	GameSessions = make(map[string]int)
	//s := http.StripPrefix("/files/", http.FileServer(http.Dir("./files/")))
	router := web.New(Context{}).Middleware(web.LoggerMiddleware).Middleware((*Context).ErrorHandler)
	router.Subrouter(Context{}, "/").Post("/reg", (*Context).Reg)
	router.Subrouter(Context{}, "/").Post("/auth", (*Context).Auth)
	router.Subrouter(Context{}, "/").Post("/newGame", (*Context).NewGame)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Middleware((*Context).LoadHero).Post("/connect", (*Context).Connect)
	router.Subrouter(Context{}, "/").Post("/check", (*Context).CheckGameSession)
	router.Subrouter(Context{}, "/").Post("/heroList", (*Context).GetHeroes)
	//router.Subrouter(Context{}, "/files").Get("/", http.Handle(s))

	fmt.Println("Запускаемся. Слушаем порт 8080")
	http.Handle("/", router)
	//http.Handle("/files/", s)
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
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newPlayer sPlayerInfo
	err := buf.Decode(&newPlayer)

	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		log.Printf(err.Error())
		return
	}
	user := []sUser{}
	err = Conn.Select(&user, "select * from users where login=? and session=?", newPlayer.Login, newPlayer.Session)

	if len(user) != 1 {
		c.SetError(401, "Неверный логин или ключ сессии. Нужна повторная авторизация")
		return
	}

	//fmt.Println(user[0].RoleId)
	if user[0].RoleId != 1 {
		c.SetError(403, "Новую игру может начать только GameMaster")
		return
	}

	session, err := uuid.NewV4()
	if err != nil {
		c.SetError(500, "Не удалось создать игру")
		log.Printf(err.Error())
		return
	}

	_, err = Conn.Exec("update users set game=? where id=?", session, user[0].ID)
	if err != nil {
		c.SetError(500, "Невозможно подключиться к игровой сессии")
		return
	}

	var game sGame
	game.Player = make([]sPlayer, 1, 1)
	game.Player[0].Login = newPlayer.Login
	game.Player[0].Id = user[0].ID
	game.Player[0].Hero = nil
	game.Count = 1
	GameMap[session.String()] = game
	//GameSessions = append(GameSessions, session.String())
	GameSessions[session.String()] = 1

	c.Response = session.String()
	return
}

func (c *Context) CheckUserSession(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newPlayer sConnectPlayer
	err := buf.Decode(&newPlayer)
	//fmt.Println(newPlayer.PlayerInfo.Login)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		log.Printf(err.Error())
		return
	}

	user := []sUser{}
	err = Conn.Select(&user, "select * from users where login=? and session=?", newPlayer.PlayerInfo.Login, newPlayer.PlayerInfo.Session)

	if len(user) != 1 {
		c.SetError(401, "Неверный логин или ключ сессии. Нужна повторная авторизация")
		return
	}
	c.User = &user[0]
	c.Player = &newPlayer
	next(iWrt, iReq)
}

func (c *Context) Connect(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.Player.Hero == 0 {
		c.SetError(404, "Невозможно найти героя")
		return
	}

	if c.User.RoleId != 2 {
		c.SetError(403, "Подключиться к сессии может только player")
		return
	}
	gameSession := c.Player.Game

	n := 0
	if _, ok := GameMap[gameSession]; ok {
		if c.User.Game == gameSession {
			for _, i := range GameMap[gameSession].Player {
				if i.Id == /*user[0].ID*/ c.User.ID {
					break
				}
				n++
			}
			c.Response = fmt.Sprintf("%d", n+1)
			return
		}
	} else {
		c.SetError(404, "Игровая сессия не найдена")
		return
	}
	if GameMap[gameSession].Count == maxCount {
		c.SetError(403, "Подключиться к сессии не удалось. Сессия заполнена")
		return
	}
	//next(iWrt, iReq)
	if c.Err != nil {
		return
	}
	var Player sPlayer
	Player.Login = c.User.Login
	Player.Id = c.User.ID
	//Player.Hero = new(sHero)
	/*err := Player.Hero.LoadHero(c.Player.Hero)
	if err != nil {
		c.SetError(500, "Невозможно загрузить героя")
		return
	}*/
	Player.Hero = c.Hero
	if game, ok := GameMap[gameSession]; ok {
		_, err := Conn.Exec("update users set game=? where id=?", gameSession, c.User.ID)
		if err != nil {
			c.SetError(500, "Невозможно подключиться к игровой сессии")
			return
		}
		game.Count++
		game.Player = append(game.Player, Player)
		GameMap[gameSession] = game
		fmt.Println(GameMap[gameSession].Count)
		if GameMap[gameSession].Count == maxCount {
			delete(GameSessions, gameSession)
		}
		c.Response = fmt.Sprintf("%d", len(GameMap[gameSession].Player))
		/*for _, i := range GameMap[session].Player {
			fmt.Println(i.PlayerInfo.Login)
		}*/
		return
	} else {
		c.SetError(404, "Игровая сессия не найдена")
	}
	return
}

func (c *Context) GetHeroes(iWrt web.ResponseWriter, iReq *web.Request) {
	list := make([]HeroToShow, 0, 0)
	LoadHeroList(2, &list)
	return
}

func LoadHeroList(idUser int, h *[]HeroToShow) (err error) {
	h = nil
	heroes := []HeroToShow{}
	err = Conn.Select(&heroes, "select heroes.id, name from herotouser inner join heroes on idhero = heroes.id where iduser = ?", idUser)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	buf, err := json.Marshal(heroes)
	if err != nil {
		return err
	}
	log.Println(string(buf))
	h = &heroes
	return nil
}

func (c *Context) LoadHero(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) { //TODO добавить функцию поиска героя
	hero := []sHeroDB{}
	err := Conn.Select(&hero, "select * from Heroes where id=?", c.Player.Hero)
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
	next(iWrt, iReq)
}

func (h *sHero) LoadWeapons() error {
	h.Weapons = nil
	weapons := []sWeaponDB{}
	err := Conn.Select(&weapons, "SELECT weapons.Id, weapons.Name, weapons.Damage, dmgtype.name as 'DmgType', weapontype.Name as 'WeaponType', weapons.Cost, weapons.Weight from weapons inner join dmgtype on weapons.dmgtype = dmgtype.id inner join weapontype on weapons.Type= weapontype.id where weapons.Id = ? and weapons.Id = ?", h.HeroDB.WeaponFirstId, h.HeroDB.WeaponSecondId)
	if err != nil {
		return err
	}
	h.Weapons = weapons
	return nil
}

func (c *Context) CheckGameSession(iWrt web.ResponseWriter, iReq *web.Request) {
	buf := json.NewDecoder(iReq.Body)
	defer iReq.Body.Close()

	var newPlayer sPlayerInfo
	err := buf.Decode(&newPlayer)
	fmt.Println(newPlayer.Login)
	if err != nil {
		c.SetError(400, "Невозможно преобразовать тело запроса в json")
		log.Printf(err.Error())
		return
	}
	user := []sUser{}
	err = Conn.Select(&user, "select * from users where login=? and session=?", newPlayer.Login, newPlayer.Session)

	if len(user) != 1 {
		c.SetError(401, "Неверный логин или ключ сессии. Нужна повторная авторизация")
		return
	}

	c.Response = user[0].Game
	return
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
		c.Response = user.Session
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
		lData, err := json.Marshal(c.Err) //добавить структуру
		if err != nil {
			iWrt.WriteHeader(500)
			fmt.Fprintln(iWrt, "")
		}
		fmt.Fprintln(iWrt, string(lData))
	} else {
		lData, err := json.Marshal(c) //добавить структуру
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
