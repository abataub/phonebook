package main

import (
	"fmt"
	"net/http"
	"time"
)

type session struct {
	Token     string         // this is the md5 hash, unique id
	Username  string         // associated username
	Firstname string         // user's first name
	UID       int            // user's db uid
	Urole     Role           // user's role for permissions
	CoCode    int            // logged in user's company
	ImageURL  string         // user's picture
	Expire    time.Time      // when does the cookie expire
	Pp        map[string]int // quick way to reference person permissions based on field name
	Pco       map[string]int // quick way to reference company permissions based on field name
	Pcl       map[string]int // quick way to reference class permissions based on field name
}

var sessions map[string]*session

func sessionInit() {
	sessions = make(map[string]*session)
}
func sessionGet(token string) *session {
	return sessions[token]
}

func (s *session) ToString() string {
	if nil == s {
		return "nil"
	}
	return fmt.Sprintf("User(%s) Name(%s) UID(%d) Token(%s)  Role(%s)",
		s.Username, s.Firstname, s.UID, s.Token, s.Urole.Name)
}

func dumpSessions() {
	i := 0
	for _, v := range sessions {
		fmt.Printf("%2d. %s\n", i, v.ToString())
		i++
	}
}

func hasPERMMODaccess(token string, fieldName string) bool {
	fmt.Printf("hasPERMMODaccess: token = %s, looking for fieldName = %s  PERMMOD = ", token, fieldName)
	s := sessions[token]
	s, ok := sessions[token]
	if !ok {
		fmt.Println("false (session not found)")
		fmt.Printf("hasPERMMODaccess:  Could not find session for %s\n", token)
		return false
	}

	for i := 0; i < len(s.Urole.Perms); i++ {
		dulog("hasPERMMODaccess: field = %s\n", s.Urole.Perms[i].Field)
		if s.Urole.Perms[i].Field == fieldName {
			ok = (0 != s.Urole.Perms[i].Perm&PERMMOD)
			dulog("%v\n", ok)
			return ok // could be true or false
		}
	}
	dulog("false (field not found)")
	return false
}

func getRoleInfo(rid int, s *session) {
	found := -1
	idx := -1

	// try to find the requested index
	dulog("len(Phonebook.Roles)=%d\n", len(Phonebook.Roles))
	dulog("getRoleInfo - looking for rid=%d\n", rid)
	for i := 0; i < len(Phonebook.Roles); i++ {
		dulog("Phonebook.Roles[%d] = %+v\n", i, Phonebook.Roles[i])
		if rid == Phonebook.Roles[i].RID {
			found = i
			idx = i
			s.Urole.Name = Phonebook.Roles[i].Name
			s.Urole.RID = rid
			break
		}
	}

	if found < 0 {
		idx = 0
		dulog("Did not find rid == %d, all permissions set to read-only\n", rid)
	}

	r := Phonebook.Roles[idx]
	//s.Urole.Perms = make([]FieldPerm, len(r.Perms))
	s.Pp = make(map[string]int)
	s.Pco = make(map[string]int)
	s.Pcl = make(map[string]int)

	for i := 0; i < len(r.Perms); i++ {
		var f FieldPerm
		f.Elem = r.Perms[i].Elem
		f.Field = r.Perms[i].Field
		if found < 0 {
			f.Perm = PERMVIEW
		} else {
			f.Perm = r.Perms[i].Perm
		}
		s.Urole.Perms = append(s.Urole.Perms, f)

		// fast access maps:
		switch f.Elem {
		case ELEMPERSON:
			s.Pp[f.Field] = f.Perm
		case ELEMCOMPANY:
			s.Pco[f.Field] = f.Perm
		case ELEMCLASS:
			s.Pcl[f.Field] = f.Perm
		}
	}

}

func sessionNew(token, username, firstname string, uid int, rid int, image string) *session {
	s := new(session)
	s.Token = token
	s.Username = username
	s.Firstname = firstname
	s.UID = uid
	s.ImageURL = image
	getRoleInfo(rid, s)

	if Phonebook.Debug {
		for i := 0; i < len(s.Urole.Perms); i++ {
			ulog("f: %s,  perm: %02x\n", s.Urole.Perms[i].Field, s.Urole.Perms[i].Perm)
		}
	}

	var d personDetail
	d.UID = uid
	//getSecurityList(&d)
	err := Phonebook.db.QueryRow("select cocode from people where uid=?", uid).Scan(&s.CoCode)
	if nil != err {
		ulog("Unable to read CoCode for userid=%d,  err = %v\n", uid, err)
	}

	sessions[token] = s
	dulog("New Session: %s\n", s.ToString())
	dulog("session.Urole.perms = %+v\n", s.Urole.Perms)
	return s
}

func (s *session) refresh(w http.ResponseWriter, r *http.Request) int {
	cookie, err := r.Cookie("accord")
	if nil != cookie && err == nil {
		cookie.Expires = time.Now().Add(10 * time.Minute)
		s.Expire = cookie.Expires
		cookie.Path = "/"
		http.SetCookie(w, cookie)
		return 0
	}
	return 1
}

//=====================================================================================
// elemPermsAny determines whether or not the session has permissions to perform the
// requested operations.  NOTE:  This interface does check the UID to fully cover
// permissions PERMOWNERVIEW or PERMOWNERMOD. This must be done at a higher level.
//
// ARGS:
//   ent  = which element? ELEMPERSON, ELEMCOMPANY, ELEMCLASS, ...
//   perm = logical or of the desired permissions.  Example PERMCREATE | PERMMOD
//
// RETURNS:
//   true if there are ANY fields for the specified element for
//   with the requested permission.
//=====================================================================================
func (s *session) elemPermsAny(elem int, perm int) bool {
	dulog("elemPermsAny:  elem=%d, perm = 0x%02x\n", elem, perm)
	for i := 0; i < len(s.Urole.Perms); i++ {
		dulog("s.Urole.Perms[%d].Elem = %d\n", i, s.Urole.Perms[i].Elem)
		if s.Urole.Perms[i].Elem == elem {
			res := s.Urole.Perms[i].Perm & perm
			dulog("fieldname: %s  s.Urole.Perms[%d].Perm = 0x%02x, s.Urole.Perms[%d].Perm & perm = 0x%02x\n",
				s.Urole.Perms[i].Field, i, s.Urole.Perms[i].Elem, i, res)
			if res != 0 { // if any of the permissions exist
				dulog("return true") // we're good to go for this check
				return true
			}
		}
	}
	dulog("return false")
	return false
}

//=====================================================================================
// elemPermsAll determines whether or not the session has permissions to perform the
// requested operations.  NOTE: This interface does check the UID to fully cover
// permissions PERMOWNERVIEW or PERMOWNERMOD. This must be done at a higher level.
//
// ARGS:
//   ent  = which element? ELEMPERSON, ELEMCOMPANY, ELEMCLASS, ...
//   perm = logical or of the desired permissions.  Example PERMCREATE | PERMMOD
//
// RETURNS:
//   true if ALL permission fields for the specified element are present
//=====================================================================================
func (s *session) elemPermsAll(elem int, perm int) bool {
	dulog("elemPermsAll:  elem=%d, perm = 0x%02x\n", elem, perm)
	for i := 0; i < len(s.Urole.Perms); i++ {
		dulog("s.Urole.Perms[%d].Elem = %d\n", i, s.Urole.Perms[i].Elem)
		if s.Urole.Perms[i].Elem == elem {
			res := s.Urole.Perms[i].Perm & perm
			dulog("fieldname: %s  s.Urole.Perms[%d].Perm = 0x%02x, s.Urole.Perms[%d].Perm & perm = 0x%02x\n",
				s.Urole.Perms[i].Field, i, s.Urole.Perms[i].Elem, i, res)
			if res == perm { // if all bits are present, res will match perm
				dulog("return true") // we're good to go for this check
				return true
			}
		}
	}
	dulog("return false")
	return false
}

// remove the supplied session.
// if there is a better idiomatic way to do this, please let me know.
func sessionDelete(s *session) {
	// fmt.Printf("Session being deleted: %s\n", s.ToString())
	// fmt.Printf("sessions before delete:\n")
	// dumpSessions()

	ss := make(map[string]*session, 0)
	for k, v := range sessions {
		if s.Token != k {
			ss[k] = v
		}
	}
	sessions = ss
	// fmt.Printf("sessions after delete:\n")
	// dumpSessions()
}