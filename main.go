package main

import (
	"fmt"
	"io/ioutil"

	//"database/sql"
	"log"
	"net/http"

	//"bytes"
	"encoding/json"

	//"strings"

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
	Err      *httpError `json:"error,omitempty"`
	Response string     `json:"response,omitempty"`
	Data     string     `json:"data,omitempty"`
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
}

type sPlayer struct {
	PlayerInfo sPlayerInfo
	Hero       *sHero
}

type sGame struct {
	Player []sPlayer
	Count  int
}

type sHero struct {
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
	/*Conn, err = sqlx.Connect("mysql", config.DBConnect)
	if err != nil {
		log.Printf(err.Error())
		return
	}*/
	//s := http.StripPrefix("/files/", http.FileServer(http.Dir("./files/")))
	router := web.New(Context{}).Middleware(web.LoggerMiddleware).Middleware((*Context).ErrorHandler)
	router.Subrouter(Context{}, "/").Post("/reg", (*Context).Reg)
	router.Subrouter(Context{}, "/").Post("/auth", (*Context).Auth)
	router.Subrouter(Context{}, "/").Post("/newGame", (*Context).NewGame)
	router.Subrouter(Context{}, "/").Post("/connect", (*Context).Connect)
	router.Subrouter(Context{}, "/").Post("/check", (*Context).CheckGameSession)
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
	game.Player[0].PlayerInfo = newPlayer
	game.Player[0].Hero = nil
	game.Count = 1
	GameMap[session.String()] = game
	//GameSessions = append(GameSessions, session.String())
	GameSessions[session.String()] = 1

	c.Response = session.String()
	return
}

func (c *Context) Connect(iWrt web.ResponseWriter, iReq *web.Request) {
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

	if user[0].RoleId != 2 {
		c.SetError(403, "Подключиться к сессии может только player")
		return
	}
	session := newPlayer.Game
	if GameMap[session].Count == maxCount {
		c.SetError(403, "Подключиться к сессии не удалось. Сессия заполнена")
		return
	}
	n := 0
	if user[0].Game == session {
		for _, i := range GameMap[session].Player {
			if i.PlayerInfo.Login == user[0].Login {
				i.PlayerInfo.Session = user[0].Session
			}
			n++
		}
		c.Response = fmt.Sprintf("%d", n)
		return
	}
	var Player sPlayer
	Player.PlayerInfo = newPlayer.PlayerInfo
	Player.Hero = nil
	if game, ok := GameMap[session]; ok {
		_, err = Conn.Exec("update users set game=? where id=?", session, user[0].ID)
		if err != nil {
			c.SetError(500, "Невозможно подключиться к игровой сессии")
			return
		}
		game.Count++
		game.Player = append(game.Player, Player)
		GameMap[session] = game
		fmt.Println(GameMap[session].Count)
		if GameMap[session].Count == maxCount {
			delete(GameSessions, session)
		}
		c.Response = fmt.Sprintf("%d", len(GameMap[session].Player))
		/*for _, i := range GameMap[session].Player {
			fmt.Println(i.PlayerInfo.Login)
		}*/
		return
	} else {
		c.SetError(404, "Игровая сессия не найдена")
	}
	return

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
	} /*else {
		iWrt.WriteHeader(200) //возможно сам пишет
	}*/
	lData, err := json.Marshal(c) //добавить структуру
	if err != nil {
		iWrt.WriteHeader(500)
		fmt.Fprintln(iWrt, "")
	}
	fmt.Fprintln(iWrt, string(lData))
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

/*func (c *Context) SendStatus(code int, text string, iWrt web.ResponseWriter) {
	if c.Err != nil {
		iWrt.WriteHeader(c.Err.Code)
	} else {
		iWrt.WriteHeader(200)
	}
	lData, err := json.Marshal(c)
	if err != nil {
		fmt.Fprintln(iWrt, "Все пропало")
		return
	}
	fmt.Println(string(lData))
	fmt.Fprintln(iWrt, string(lData))
}*/

/*func (c *Context) FilesHandler(iWrt web.ResponseWriter, iReq *web.Request) {
	http.FileServer(http.Dir("./files"))
}*/
