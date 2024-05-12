package database

import (
	"database/sql"
	"dependency-confusion/internal/models"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

func InitDB() (*sql.DB, error) {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	
	db, err := sql.Open("sqlite3", filepath.Join(basepath, "packages.db"))
	if err != nil {
		return nil, err
	}
	
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS `packages` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `timestamp` DATETIME DEFAULT CURRENT_TIMESTAMP, `package` TEXT NOT NULL, `source_ip` TEXT NOT NULL, `hostname` TEXT NOT NULL, `username` TEXT NOT NULL, `cwd` TEXT NOT NULL)")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func AddData(db *sql.DB, sourceIP string, data *models.DataExfiltrated) error {

	_, err := db.Exec("insert into packages (timestamp, package, source_ip, hostname, username, cwd) values ($1, $2, $3, $4, $5, $6)",
		time.Now(), data.Package, sourceIP, data.Hostname, data.Username, data.CWD)
	if err != nil {
		return err
	}

	return nil
}

func GetData(db *sql.DB) ([]*models.DbPackage, error) {
	rows, err := db.Query("select * from packages")
	if err != nil {
		return []*models.DbPackage{}, err
	}
	defer rows.Close()

	pkg := []*models.DbPackage{}

	for rows.Next() {
		p := models.DbPackage{}
		err := rows.Scan(&p.Id, &p.Timestamp, &p.DataExfiltrated.Package, &p.SourceIP, &p.DataExfiltrated.Hostname, &p.DataExfiltrated.Username, &p.DataExfiltrated.CWD)
		if err != nil {
			log.Warnln(err)
			continue
		}
		pkg = append(pkg, &p)
	}
	return pkg, nil
}
