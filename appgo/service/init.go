package service

import (
	"github.com/i2eco/ecology/appgo/dao"
	"github.com/i2eco/ecology/appgo/pkg/mus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

func Init() error {
	dao.Global = dao.NewGlobal()
	dao.DocumentSearchResult = dao.NewDocumentSearchResult()
	InitGen()
	dao.ReadRecord.UpdateReadingRule()
	dao.GithubApi = dao.NewGithubApi()
	InitMailer()

	if viper.GetBool("github.isExec") {
		go func() {
			for {
				err := dao.GithubApi.All()
				if err != nil {
					mus.Logger.Error("github api error", zap.Error(err))
				}
				time.Sleep(5 * time.Second)
			}
		}()
	}

	return nil
}
