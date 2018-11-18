package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

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
	PlayerId   int         `json:"playerId"`
}

type sPlayer struct {
	Login string
	Id    int
	Hero  *sHero
}

type sGame struct {
	Player []sPlayer
	Count  int
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

type sGameMap struct {
	m map[string]sGame
	sync.RWMutex
}

var config sConfig
var Conn *sqlx.DB

var GameMap sGameMap

//var GameMap map[string]sGame
var GameSessions map[string]int

func main() {
	err := LoadConfig()
	if err != nil {
		log.Printf(err.Error())
		return
	}
	GameMap.m = make(map[string]sGame)
	GameSessions = make(map[string]int)
	//router := web.New(Context{}).Middleware(web.LoggerMiddleware).Middleware((*Context).ErrorHandler)
	router := web.New(Context{}).Middleware((*Context).Logger).Middleware((*Context).ErrorHandler)
	router.Subrouter(Context{}, "/").Post("/reg", (*Context).Reg)
	router.Subrouter(Context{}, "/").Post("/auth", (*Context).Auth)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Post("/newGame", (*Context).NewGame).Delete("/newGame", (*Context).DestroyGame)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Middleware((*Context).Reconnect).Middleware((*Context).LoadHero).Post("/connect", (*Context).Connect)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Delete("/connect", (*Context).Disconnect)
	router.Subrouter(Context{}, "/").Post("/heroList", (*Context).GetHeroes)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Post("/SaveHero", (*Context).SaveHero)
	router.Subrouter(Context{}, "/").Middleware((*Context).CheckUserSession).Post("/SaveGame", (*Context).SaveGame)

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
	game.Player = make([]sPlayer, maxCount)
	game.Player[0].Login = c.User.Login
	game.Player[0].Id = c.User.ID
	game.Player[0].Hero = nil
	game.Count = 1
	GameMap.Lock()
	GameMap.m[session.String()] = game
	defer GameMap.Unlock()
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
	return
}

func (c *Context) Reconnect(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	gameSession := c.User.Game
	n := 0
	if _, ok := GameMap.m[gameSession]; ok {
		for _, i := range GameMap.m[gameSession].Player {
			if i.Id == c.User.ID {
				break
			}
			n++
		}
		if n <= maxCount {
			c.Hero = GameMap.m[gameSession].Player[n].Hero
			c.Response = fmt.Sprintf("%d", n)
		} else {
			_, err := Conn.Exec("update users set game=? where id=?", "", c.User.ID)
			if err != nil {
				c.SetError(500, "Ошибка БД")
				return
			}
			c.SetError(403, "Невозможно переподключиться к игровой сессии")
		}
		return
	} else {
		//log.Println("Будем подключаться к новой сессии")
		next(iWrt, iReq)
		return
	}
	return
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

	if GameMap.m[gameSession].Count == maxCount {
		c.SetError(403, "Подключиться к сессии не удалось. Сессия заполнена")
		return
	}

	if c.Err != nil {
		return
	}

	GameMap.Lock()
	defer GameMap.Unlock()
	var Player sPlayer
	Player.Login = c.User.Login
	Player.Id = c.User.ID

	Player.Hero = c.Hero
	if game, ok := GameMap.m[gameSession]; ok {
		_, err := Conn.Exec("update users set game=? where id=?", gameSession, c.User.ID)
		if err != nil {
			c.SetError(500, "Невозможно подключиться к игровой сессии")
			return
		}
		n := 1
		for ; n < len(game.Player); n++ {
			if game.Player[n].Hero == nil {
				break
			}
		}
		game.Count++
		//game.Player = append(game.Player, Player)
		game.Player[n] = Player
		GameMap.m[gameSession] = game
		if GameMap.m[gameSession].Count == maxCount {
			delete(GameSessions, gameSession)
		}
		c.Response = fmt.Sprintf("%d", n)
		/*for _, i := range GameMap[session].Player {
			fmt.Println(i.PlayerInfo.Login)
		}*/
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
	n := 0
	GameMap.Lock()
	defer GameMap.Unlock()
	gameSession := c.User.Game
	if _, ok := GameMap.m[c.User.Game]; ok {
		_, err := Conn.Exec("Update users set game = '' where id=?", c.User.ID)
		if err != nil {
			c.SetError(500, "Ошибка при удалении игры")
			return
		}
		for _, i := range GameMap.m[gameSession].Player {
			if i.Id == c.User.ID {
				break
			}
			n++
		}
		game := GameMap.m[c.User.Game]
		game.Player[n].Hero = nil
		game.Player[n].Id = 0
		game.Player[n].Login = ""
		game.Count--
		GameMap.m[c.User.Game] = game
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
	//c.Hero.Weapons = make([]sWeaponDB, 2)
	c.Hero.HeroDB = &hero[0]
	c.Hero.LoadWeapons()
	next(iWrt, iReq)
}

func (h *sHero) LoadWeapons() error {
	h.Weapons = make([]sWeaponDB, 2)
	weapons := []sWeaponDB{}
	err := Conn.Select(&weapons, "SELECT weapons.Id, weapons.Name, weapons.Damage, dmgtype.name as 'DmgType', weapontype.Name as 'WeaponType', weapons.Cost, weapons.Weight from weapons inner join dmgtype on weapons.dmgtype = dmgtype.id inner join weapontype on weapons.Type= weapontype.id where weapons.Id = ? and weapons.Id = ?", h.HeroDB.WeaponFirstId, h.HeroDB.WeaponSecondId)
	if err != nil {
		return err
	}
	h.Weapons = weapons
	return nil
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

func BoolToInt(b bool) byte {
	if b {
		return byte(1)
	}
	return byte(0)
}

func (c *Context) SaveHero(iWrt web.ResponseWriter, iReq *web.Request) {
	if c.User.RoleId != 2 {
		c.SetError(403, "Сохранить героя может только игрок")
		return
	}
	if game, ok := GameMap.m[c.User.Game]; ok {
		if game.Player[c.Player.PlayerId].Id == c.User.ID {
			h := game.Player[c.Player.PlayerId].Hero.HeroDB
			_, err := Conn.Exec("update heroes set Exp=100, Speed=?, HP=?, HPmax=?, HitBonesMax=?, HitBones=?, Strength=?, Perception=?, Endurance=?, Charisma=?, Intelligence=?, Agility=?, MasterBonus=?, DeathSavingThrowGood=?, DeathSavingThrowBad=?, TemporaryHP=?, AC=?, Initiative=?, PassiveAttention=?, Inspiration=?, Ammo=?, Languages=?, SavingThrowS=?, SavingThrowP=?, SavingThrowE=?, SavingThrowC=?, SavingThrowI=?, SavingThrowA=?, Athletics=?, Acrobatics=?, Juggle=?, Stealth=?, Magic=?, History=?, Analysis=?, Nature=?, Religion=?, AnimalCare=?, Insight=?, Medicine=?, Attention=?, Survival=?, Deception=?, Intimidation=?, Performance=?, Conviction=?, WeaponFirstId=?, WeaponSecondId=?, ArmorId=?, ShieldId=? where Id=?", h.Speed, h.HP, h.HPmax, h.HitBonesMax, h.HitBones, h.Strength, h.Perception, h.Endurance, h.Charisma, h.Intelligence, h.Agility, h.MasterBonus, h.DeathSavingThrowGood, h.DeathSavingThrowBad, h.TemporaryHP, h.AC, h.Initiative, h.PassiveAttention, h.Inspiration, h.Ammo, h.Languages, h.SavingThrowS, h.SavingThrowP, h.SavingThrowE, h.SavingThrowC, h.SavingThrowI, h.SavingThrowA, h.Athletics, h.Acrobatics, h.Juggle, h.Stealth, h.Magic, h.History, h.Analysis, h.Nature, h.Religion, h.AnimalCare, h.Insight, h.Medicine, h.Attention, h.Survival, h.Deception, h.Intimidation, h.Performance, h.Conviction, h.WeaponFirstId, h.WeaponSecondId, h.ArmorId, h.ShieldId, h.Id)

			if err != nil {
				c.SetError(500, "Невозможно сохранить героя")
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
	if c.User.RoleId != 1 {
		c.SetError(403, "Сохранить игру может только мастер")
		return
	}
	if game, ok := GameMap.m[c.User.Game]; ok {
		var e error
		tx, err := Conn.Begin()
		if err != nil {
			c.SetError(500, "Невозможно начать транзакцию с БД")
			return
		}
		for _, i := range game.Player {
			if i.Hero != nil {
				h := i.Hero.HeroDB
				_, err := tx.Exec("update heroes set Exp=100, Speed=?, HP=?, HPmax=?, HitBonesMax=?, HitBones=?, Strength=?, Perception=?, Endurance=?, Charisma=?, Intelligence=?, Agility=?, MasterBonus=?, DeathSavingThrowGood=?, DeathSavingThrowBad=?, TemporaryHP=?, AC=?, Initiative=?, PassiveAttention=?, Inspiration=?, Ammo=?, Languages=?, SavingThrowS=?, SavingThrowP=?, SavingThrowE=?, SavingThrowC=?, SavingThrowI=?, SavingThrowA=?, Athletics=?, Acrobatics=?, Juggle=?, Stealth=?, Magic=?, History=?, Analysis=?, Nature=?, Religion=?, AnimalCare=?, Insight=?, Medicine=?, Attention=?, Survival=?, Deception=?, Intimidation=?, Performance=?, Conviction=?, WeaponFirstId=?, WeaponSecondId=?, ArmorId=?, ShieldId=? where Id=?", h.Speed, h.HP, h.HPmax, h.HitBonesMax, h.HitBones, h.Strength, h.Perception, h.Endurance, h.Charisma, h.Intelligence, h.Agility, h.MasterBonus, h.DeathSavingThrowGood, h.DeathSavingThrowBad, h.TemporaryHP, h.AC, h.Initiative, h.PassiveAttention, h.Inspiration, h.Ammo, h.Languages, h.SavingThrowS, h.SavingThrowP, h.SavingThrowE, h.SavingThrowC, h.SavingThrowI, h.SavingThrowA, h.Athletics, h.Acrobatics, h.Juggle, h.Stealth, h.Magic, h.History, h.Analysis, h.Nature, h.Religion, h.AnimalCare, h.Insight, h.Medicine, h.Attention, h.Survival, h.Deception, h.Intimidation, h.Performance, h.Conviction, h.WeaponFirstId, h.WeaponSecondId, h.ArmorId, h.ShieldId, h.Id)
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

func (c *Context) Logger(iWrt web.ResponseWriter, iReq *web.Request, next web.NextMiddlewareFunc) {
	t := time.Now()
	next(iWrt, iReq)
	if c.Err != nil {
		fmt.Printf("[ %s ] %d %s\n", time.Since(t), 200, iReq.URL)
		return
	}
	fmt.Printf("[ %s ] %d %s\n", time.Since(t), 200, iReq.URL)
}
