package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/beego"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/i2eco/ecology/appgo/dao"
	"github.com/i2eco/ecology/appgo/model/mysql"
	"github.com/i2eco/ecology/appgo/pkg/code"
	"github.com/i2eco/ecology/appgo/pkg/conf"
	"github.com/i2eco/ecology/appgo/pkg/mus"
	"github.com/i2eco/ecology/appgo/pkg/utils"
	"github.com/i2eco/ecology/appgo/router/core"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"html/template"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 解析并提取版本控制的commit内容
func parseGitCommit(str string) (cont, commit string) {
	var slice []string
	arr := strings.Split(str, "<bookstack-git>")
	if len(arr) > 1 {
		slice = append(slice, arr[0])
		str = strings.Join(arr[1:], "")
	}
	arr = strings.Split(str, "</bookstack-git>")
	if len(arr) > 1 {
		slice = append(slice, arr[1:]...)
		commit = arr[0]
	}
	if len(slice) > 0 {
		cont = strings.Join(slice, "")
	} else {
		cont = str
	}
	return
}

//判断用户是否可以阅读文档.
func isReadable(c *core.Context, identify, token string) (resp *mysql.BookResult) {
	book, err := dao.Book.FindByFieldFirst("identify", identify)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.Html404()
		return
	}

	//如果文档是私有的
	if book.PrivatelyOwned == 1 && !c.Member().IsAdministrator() {
		isOk := false
		if c.Member() != nil {
			_, err := dao.Relationship.FindForRoleId(book.BookId, c.Member().MemberId)
			if err == nil {
				isOk = true
			}
		}

		if book.PrivateToken != "" && !isOk {
			//如果有访问的Token，并且该项目设置了访问Token，并且和用户提供的相匹配，则记录到Session中.
			//如果用户未提供Token且用户登录了，则判断用户是否参与了该项目.
			//如果用户未登录，则从Session中读取Token.
			if token != "" && strings.EqualFold(token, book.PrivateToken) {
				c.SetSession(identify, token)
			} else if token, ok := c.GetSession(identify).(string); !ok || !strings.EqualFold(token, book.PrivateToken) {
				hasErr := ""
				if c.Context.Request.Method == "POST" {
					hasErr = "true"
				}
				c.Redirect(302, beego.URLFor("DocumentController.Index", ":key", identify)+"?with-password=true&err="+hasErr)
				return
			}
		} else if !isOk {
			c.Html404()
			return
		}
	}

	bookResult := book.ToBookResult()
	if c.Member() != nil {
		rel, err := dao.Relationship.FindByBookIdAndMemberId(bookResult.BookId, c.Member().MemberId)
		if err == nil {
			bookResult.MemberId = rel.MemberId
			bookResult.RoleId = rel.RoleId
			bookResult.RelationshipId = rel.RelationshipId
		}
	}
	//判断是否需要显示评论框
	switch bookResult.CommentStatus {
	case "closed":
		bookResult.IsDisplayComment = false
	case "open":
		bookResult.IsDisplayComment = true
	case "group_only":
		bookResult.IsDisplayComment = bookResult.RelationshipId > 0
	case "registered_only":
		bookResult.IsDisplayComment = true
	}
	return bookResult
}

//文档首页.
func Index(c *core.Context) {
	identify := c.Param("key")
	if identify == "" {
		c.Html404()
		return
	}

	token, _ := c.GetQuery("token")
	withPwd, _ := c.GetQuery("with-password")
	tab, _ := c.GetQuery("tab")
	if len(strings.TrimSpace(withPwd)) > 0 {
		indexWithPassword(c)
		return
	}

	tab = strings.ToLower(tab)

	bookResult := isReadable(c, identify, token)
	if bookResult.BookId == 0 { //没有阅读权限
		c.Redirect(302, "/")
		return
	}

	bookResult.Lang = utils.GetLang(bookResult.Lang)
	c.Tpl().Data["Book"] = bookResult

	switch tab {
	case "comment", "score":
	default:
		tab = "default"
	}
	c.Tpl().Data["Qrcode"] = dao.Member.GetQrcodeByUid(bookResult.MemberId)
	c.Tpl().Data["MyScore"] = dao.Score.BookScoreByUid(c.Member().MemberId, bookResult.BookId)
	c.Tpl().Data["Tab"] = tab
	if beego.AppConfig.DefaultBool("showWechatCode", false) && bookResult.PrivatelyOwned == 0 {
		wechatCode := mysql.NewWechatCode()
		go wechatCode.CreateWechatCode(bookResult.BookId) //如果已经生成了小程序码，则不会再生成
		c.Tpl().Data["Wxacode"] = wechatCode.GetCode(bookResult.BookId)
	}

	//当前默认展示100条评论
	c.Tpl().Data["Comments"], _ = dao.Comments.Comments(1, 100, bookResult.BookId, 1)
	c.Tpl().Data["Menu"], _ = dao.Document.GetMenuTop(bookResult.BookId)
	title := "《" + bookResult.BookName + "》"
	if tab == "comment" {
		title = "点评 - " + title
	}
	c.GetSeoByPage("book_info", map[string]string{
		"title":       title,
		"keywords":    bookResult.Label,
		"description": bookResult.Description,
	})
	c.Tpl().Data["RelateBooks"] = mysql.NewRelateBook().Lists(bookResult.BookId)
	c.Html("document/intro")

}

//文档首页.
func indexWithPassword(c *core.Context) {
	identify := c.Param("key")
	if identify == "" {
		c.Html404()
		return
	}
	c.GetSeoByPage("book_info", map[string]string{
		"title":       "密码访问",
		"keywords":    "密码访问",
		"description": "密码访问",
	})
	c.Tpl().Data["ShowErrTips"] = c.GetString("err") != ""
	c.Tpl().Data["Identify"] = identify
	c.Html("document/read-with-password")
	return
}

//阅读文档.
func ReadHtml(c *core.Context) {
	identify := c.Param("key")
	id := c.Param("id")
	token, _ := c.GetQuery("token")

	if identify == "" {
		c.Html404()
		return
	}
	//如果没有开启你们匿名则跳转到登录
	if !dao.Global.IsEnableAnonymous() && c.Member() == nil {
		c.Redirect(302, "/login")
		return
	}

	bookResult := isReadable(c, identify, token)

	doc := mysql.NewDocument()
	var err error
	var docId int
	// 编辑的文档
	if id != "" {
		if docId, _ = strconv.Atoi(id); docId > 0 {
			doc, err = dao.Document.Find(docId) //文档id
			if err != nil {
				mus.Logger.Error("read doc find int doc id error", zap.Int("docId", docId), zap.Error(err))
				c.Html404()
				return
			}
		} else {
			//此处的id是字符串，标识文档标识，根据文档标识和文档所属的书的id作为key去查询
			doc, err = dao.Document.FindByBookIdAndDocIdentify(bookResult.BookId, id) //文档标识
			if err != nil {
				mus.Logger.Error("read doc find string doc id error", zap.String("docId", id), zap.Error(err))
				c.Html404()
				return
			}
		}

		// 查找第一篇文章
	} else {
		trees, err := dao.Document.FindDocumentTree(bookResult.BookId, 0, true)
		if err != nil {
			mus.Logger.Error(err.Error())
			c.Html404()
			return
		}

		// 取第一篇文章
		if len(trees) == 0 {
			mus.Logger.Error("trees length is 0")
			c.Html404()
			return
		}
		docId = trees[0].DocumentId
		doc, err = dao.Document.Find(docId) //文档id
		if err != nil {
			mus.Logger.Error("read doc find int doc id error2", zap.Int("docId", docId), zap.Error(err))
			c.Html404()
			return
		}
	}

	if doc.BookId != bookResult.BookId {
		c.Html404()
		return
	}

	bodyText := ""
	authHTTPS := strings.ToLower(dao.Global.GetOptionValue("AUTO_HTTPS", "false")) == "true"
	if doc.Release != "" {
		query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(doc.Release))
		if err != nil {
			mus.Logger.Error(err.Error())
		} else {
			query.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
				src, ok := contentSelection.Attr("src")
				if ok {
					if utils.StoreType == utils.StoreOss && !(strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://")) {
						src = viper.GetString("app.ossDomain") + "/" + strings.TrimLeft(src, "./")
					}
				}
				if authHTTPS {
					if srcArr := strings.Split(src, "://"); len(srcArr) > 1 {
						src = "https://" + strings.Join(srcArr[1:], "://")
					}
				}
				contentSelection.SetAttr("src", src)
				if alt, _ := contentSelection.Attr("alt"); alt == "" {
					contentSelection.SetAttr("alt", doc.DocumentName+" - 图"+fmt.Sprint(i+1))
				}
			})
			html, err := query.Find("body").Html()
			if err != nil {
				mus.Logger.Error(err.Error())
			} else {
				doc.Release = html
			}
		}
		bodyText = query.Find(".markdown-toc").Text()
	}

	attach, err := dao.Attachment.FindListByDocumentId(doc.DocumentId)
	if err == nil {
		doc.AttachList = attach
	}

	//文档阅读人次+1
	if err := dao.SetIncreAndDecre(mysql.Document{}.TableName(), "vcnt",
		fmt.Sprintf("document_id=%v", doc.DocumentId),
		true, 1,
	); err != nil {
		mus.Logger.Error(err.Error())
	}
	//项目阅读人次+1
	if err := dao.SetIncreAndDecre(mysql.Book{}.TableName(), "vcnt",
		fmt.Sprintf("book_id=%v", doc.BookId),
		true, 1,
	); err != nil {
		mus.Logger.Error(err.Error())
	}

	if c.Member().MemberId > 0 { //增加用户阅读记录
		if err := dao.ReadRecord.Add(doc.DocumentId, c.Member().MemberId); err != nil {
			mus.Logger.Error(err.Error())
		}
	}
	parentTitle := dao.Document.GetParentTitle(doc.ParentId)
	seo := map[string]string{
		"title":       doc.DocumentName + " - 《" + bookResult.BookName + "》",
		"keywords":    bookResult.Label,
		"description": beego.Substr(bodyText+" "+bookResult.Description, 0, 200),
	}

	if len(parentTitle) > 0 {
		seo["title"] = parentTitle + " - " + doc.DocumentName + " - 《" + bookResult.BookName + "》"
	}

	//SEO
	c.GetSeoByPage("book_read", seo)

	existBookmark := dao.Bookmark.Exist(c.Member().MemberId, doc.DocumentId)

	doc.Vcnt = doc.Vcnt + 1

	mysql.NewBookCounter().Increase(bookResult.BookId, true)

	tree, err := dao.Document.CreateDocumentTreeForHtml(bookResult.BookId, doc.DocumentId)

	if err != nil {
		mus.Logger.Error(err.Error())
		c.Html404()
	}

	// 查询用户哪些文档阅读了
	if c.Member().MemberId > 0 {
		modelRecord := new(mysql.ReadRecord)
		lists, cnt, _ := modelRecord.List(c.Member().MemberId, bookResult.BookId)
		if cnt > 0 {
			var readMap = make(map[string]bool)
			for _, item := range lists {
				readMap[strconv.Itoa(item.DocId)] = true
			}
			if doc, err := goquery.NewDocumentFromReader(strings.NewReader(tree)); err == nil {
				doc.Find("li").Each(func(i int, selection *goquery.Selection) {
					if id, exist := selection.Attr("id"); exist {
						if _, ok := readMap[id]; ok {
							selection.AddClass("readed")
						}
					}
				})
				tree, _ = doc.Find("body").Html()
			}
		}
	}

	if beego.AppConfig.DefaultBool("showWechatCode", false) && bookResult.PrivatelyOwned == 0 {
		wechatCode := mysql.NewWechatCode()
		go wechatCode.CreateWechatCode(bookResult.BookId) //如果已经生成了小程序码，则不会再生成
		c.Tpl().Data["Wxacode"] = wechatCode.GetCode(bookResult.BookId)
	}

	if wd, _ := c.GetQuery("wd"); strings.TrimSpace(wd) != "" {
		c.Tpl().Data["Keywords"] = dao.NewElasticSearchClient().SegWords(wd)
	}

	fmt.Println("bookResult------>", bookResult.MemberId)

	c.Tpl().Data["Bookmark"] = existBookmark
	c.Tpl().Data["Model"] = bookResult
	c.Tpl().Data["Book"] = bookResult //文档下载需要用到Book变量
	c.Tpl().Data["Result"] = template.HTML(tree)
	c.Tpl().Data["Title"] = doc.DocumentName
	c.Tpl().Data["DocId"] = doc.DocumentId
	c.Tpl().Data["Content"] = template.HTML(doc.Release)
	c.Tpl().Data["View"] = doc.Vcnt
	c.Tpl().Data["UpdatedAt"] = doc.ModifyTime.Format("2006-01-02 15:04:05")
	c.Html("document/" + bookResult.Theme + "_read")
}

//编辑文档.
func Edit(c *core.Context) {
	docId := 0 // 文档id

	identify := c.Param("key")
	if identify == "" {
		c.Html404()
		return
	}

	bookResult := mysql.NewBookResult()

	var err error
	//如果是超级管理者，则不判断权限
	if c.Member().IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
		bookResult = book.ToBookResult()
	} else {
		bookResult, err = dao.Book.ResultFindByIdentify(identify, c.Member().MemberId)
		if err != nil {
			mus.Logger.Error(err.Error())
			c.Html404()
			return
		}

		if bookResult.RoleId == conf.BookObserver {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
	}

	c.Tpl().Data["Model"] = bookResult
	r, _ := json.Marshal(bookResult)

	c.Tpl().Data["ModelResult"] = template.JS(string(r))

	c.Tpl().Data["Result"] = template.JS("[]")

	// 编辑的文档
	if id := c.Param("id"); id != "" {
		if num, _ := strconv.Atoi(id); num > 0 {
			docId = num
		} else { //字符串
			var doc = mysql.NewDocument()
			mus.Db.Where("identify=? and book_id = ?", id, bookResult.BookId).Find(doc)
			docId = doc.DocumentId
		}
	}

	trees, err := dao.Document.FindDocumentTree(bookResult.BookId, docId, true)
	if err != nil {
		mus.Logger.Error(err.Error())
	} else {
		if len(trees) > 0 {
			if jsTree, err := json.Marshal(trees); err == nil {
				c.Tpl().Data["Result"] = template.JS(string(jsTree))
			}
		} else {
			c.Tpl().Data["Result"] = template.JS("[]")
		}
	}
	c.Tpl().Data["BaiDuMapKey"] = beego.AppConfig.DefaultString("baidumapkey", "")
	//根据不同编辑器类型加载编辑器【注：现在只支持markdown】
	c.Html("document/markdown_edit_template")

}

//批量创建文档
func CreateMulti(c *core.Context) {
	bookIdStr, _ := c.GetQuery("book_id")
	bookId, _ := strconv.Atoi(bookIdStr)

	if !(c.Member().MemberId > 0 && bookId > 0) {
		c.JSONErrStr(1, "操作失败：只有项目创始人才能批量添加")
		return
	}

	var book mysql.Book
	mus.Db.Where("book_id = ? and member_id = ?", bookId, c.Member().MemberId).Find(&book)

	if book.BookId > 0 {
		content, _ := c.GetQuery("content")
		slice := strings.Split(content, "\n")
		if len(slice) > 0 {
			for _, row := range slice {
				if chapter := strings.Split(strings.TrimSpace(row), " "); len(chapter) > 1 {
					if ok, err := regexp.MatchString(`^[a-zA-Z0-9_\-\.]*$`, chapter[0]); ok && err == nil {
						i, _ := strconv.Atoi(chapter[0])
						if chapter[0] != "0" && strconv.Itoa(i) != chapter[0] { //不为纯数字
							doc := mysql.Document{
								DocumentName: strings.Join(chapter[1:], " "),
								Identify:     chapter[0],
								BookId:       bookId,
								//Markdown:     "[TOC]\n\r",
								MemberId: c.Member().MemberId,
							}
							if docId, err := dao.Document.InsertOrUpdate(mus.Db, &doc); err == nil {
								if err := dao.DocumentStore.InsertOrUpdate(mus.Db, &mysql.DocumentStore{DocumentId: int(docId), Markdown: "[TOC]\n\r\n\r"}); err != nil {
									mus.Logger.Error(err.Error())
								}
							} else {
								mus.Logger.Error(err.Error())
							}
						}

					}
				}
			}
		}
	}
	c.JSONOK()
}

//DownloadAttachment 下载附件.
func DownloadAttachment(c *core.Context) {
	identify := c.Param(":key")
	attachId, _ := strconv.Atoi(c.Param(":attach_id"))
	token := c.GetString("token")

	memberId := 0

	if c.Member() != nil {
		memberId = c.Member().MemberId
	}
	bookId := 0

	//判断用户是否参与了项目
	bookResult, err := dao.Book.ResultFindByIdentify(identify, memberId)

	if err != nil {
		//判断项目公开状态
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.Html404()
			return
		}
		//如果不是超级管理员则判断权限
		if c.Member() == nil || c.Member().Role != conf.MemberSuperRole {
			//如果项目是私有的，并且token不正确
			if (book.PrivatelyOwned == 1 && token == "") || (book.PrivatelyOwned == 1 && book.PrivateToken != token) {
				c.Html404()
				return
			}
		}

		bookId = book.BookId
	} else {
		bookId = bookResult.BookId
	}
	//查找附件
	attachment, err := dao.Attachment.Find(attachId)

	if err != nil {
		c.Html404()
		return
	}
	if attachment.BookId != bookId {
		c.Html404()
		return
	}

	c.Download(filepath.Join("./", attachment.FilePath), attachment.FileName)
}

//删除附件.
func RemoveAttachment(c *core.Context) {
	attachIdStr, _ := c.GetQuery("attach_id")
	attachId, _ := strconv.Atoi(attachIdStr)
	if attachId <= 0 {
		c.JSONErrStr(6001, "参数错误")
		return
	}

	attach, err := dao.Attachment.Find(attachId)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.JSONErrStr(6002, "附件不存在")
		return
	}

	document, err := dao.Document.Find(attach.DocumentId)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.JSONErrStr(6003, "文档不存在")
		return
	}

	if c.Member().Role != conf.MemberSuperRole {
		rel, err := dao.Relationship.FindByBookIdAndMemberId(document.BookId, c.Member().MemberId)
		if err != nil {
			mus.Logger.Error(err.Error())
			c.JSONErrStr(6004, "权限不足")
			return
		}
		if rel.RoleId == conf.BookObserver {
			c.JSONErrStr(6004, "权限不足")
			return
		}
	}

	if err = dao.Attachment.Delete(c.Context, mus.Db, attachId); err != nil {
		mus.Logger.Error(err.Error())
		c.JSONErrStr(6005, "删除失败")
	}

	os.Remove(filepath.Join("./", attach.FilePath))
	c.JSONOK(attach)
}

//获取或更新文档内容.
func ContentGet(c *core.Context) {
	identify := c.Param("key")
	_, flag := c.GetQuery("doc_id")
	var docId int
	if !flag {
		docId, _ = strconv.Atoi(c.Param("id"))
	}
	currentMember := c.Member()
	if currentMember.IsAdministrator() {
		_, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.JSONErr(code.MsgErr, err)
			return
		}
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, currentMember.MemberId)
		if err != nil || bookResult.RoleId == conf.Conf.Info.BookObserver {
			c.JSONCode(code.MsgErr)
			return
		}
	}

	if docId <= 0 {
		c.JSONCode(code.MsgErr)
		return

	}

	doc, err := dao.Document.Find(docId)

	if err != nil {
		c.JSONCode(code.MsgErr)
		return
	}
	attach, err := dao.Attachment.FindListByDocumentId(doc.DocumentId)
	if err == nil {
		doc.AttachList = attach
	}

	//为了减少数据的传输量，这里Release和Content的内容置空，前端会根据markdown文本自动渲染
	//doc.Release = ""
	//doc.Content = ""
	doc.Markdown = dao.DocumentStore.GetFiledById(doc.DocumentId, "markdown")
	c.JSONOK(doc)

}

//获取或更新文档内容.
func ContentPost(c *core.Context) {
	identify := c.Param("key")
	_, flag := c.GetQuery("doc_id")
	errMsg := code.MsgOk
	var docId int
	if !flag {
		docId, _ = strconv.Atoi(c.Param("id"))
	}
	bookId := 0
	currentMember := c.Member()
	//如果是超级管理员，则忽略权限
	if currentMember.IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.JSONErr(code.DocContentPostErr1, err)
			return
		}
		bookId = book.BookId
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, currentMember.MemberId)

		if err != nil || bookResult.RoleId == conf.Conf.Info.BookObserver {
			c.JSONOK(code.DocContentPostErr2)
			return
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		c.JSONCode(code.DocContentPostErr3)
		return

	}

	//更新文档内容
	markdown := strings.TrimSpace(c.PostForm("markdown"))
	content := c.PostForm("html")

	// 文档拆分
	gq, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err == nil {
		seg := gq.Find("bookstack-split").Text()
		if strings.Contains(seg, "#") {
			markdown = strings.Replace(markdown, fmt.Sprintf("<bookstack-split>%v</bookstack-split>", seg), "", -1)
			err := dao.Document.SplitMarkdownAndStore(seg, markdown, docId)
			if err != nil {
				c.JSONErr(code.DocContentPostErr4, err)
				return
			}
			c.JSONOK(code.MsgOk)
			return
		}
	}

	version, _ := strconv.Atoi(c.PostForm("version"))
	isCover := c.PostForm("cover")

	doc, err := dao.Document.Find(docId)

	if err != nil {
		c.JSONErr(code.DocContentPostErr5, err)
		return
	}
	if doc.BookId != bookId {
		c.JSONCode(code.DocContentPostErr6)
		return
	}
	if doc.Version != int64(version) && !strings.EqualFold(isCover, "yes") {
		c.JSONCode(code.DocContentPostErr7)
		return
	}

	var ds = mysql.DocumentStore{}
	var actionName string

	// 替换掉<git></git>标签内容
	if markdown == "" && content != "" {
		ds.Markdown = content
	} else {
		ds.Markdown = markdown
	}

	ds.Markdown, actionName = parseGitCommit(ds.Markdown)
	ds.Content, _ = parseGitCommit(content)

	if actionName == "" {
		actionName = "--"
	}

	doc.ModifyAt = c.Member().MemberId
	doc.Version = time.Now().Unix()
	if docId, err := dao.Document.InsertOrUpdate(mus.Db, doc); err != nil {
		c.JSONErr(code.DocContentPostErr8, err)
		return
	} else {
		ds.DocumentId = int(docId)
		if err := dao.DocumentStore.InsertOrUpdate(mus.Db, &ds); err != nil {
			mus.Logger.Error(err.Error())
			c.JSONErr(code.DocContentPostErr9, err)
			return
		}
	}

	//如果启用了文档历史，则添加历史文档
	if enableDocumentHistory() > 0 {
		if len(strings.TrimSpace(ds.Markdown)) > 0 { //空内容不存储版本
			history := mysql.DocumentHistory{}
			history.DocumentId = docId
			history.DocumentName = doc.DocumentName
			history.ModifyAt = currentMember.MemberId
			history.MemberId = doc.MemberId
			history.ParentId = doc.ParentId
			history.Version = time.Now().Unix()
			history.Action = "modify"
			history.ActionName = actionName
			history.ModifyTime = time.Now()
			// todo fix

			err = dao.DocumentHistory.InsertOrUpdate(&history)
			if err != nil {
				mus.Logger.Error("DocumentHistory InsertOrUpdate => " + err.Error())
			} else {
				vc := dao.NewVersionControl(docId, history.Version)
				err = vc.SaveVersion(ds.Content, ds.Markdown)
				if err != nil {
					// todo log
				}
				err = dao.DocumentHistory.DeleteByLimit(docId, enableDocumentHistory())
				if err != nil {
					// todo log
				}
			}
		}

	}
	doc.Release = ""
	//注意：如果errMsg的值是true，则表示更新了目录排序，需要刷新，否则不刷新
	c.JSONCode(errMsg, doc)

}

//生成项目访问的二维码.
func QrCode(c *core.Context) {
	identify := c.GetString(":key")

	book, err := dao.Book.FindByIdentify(identify)

	if err != nil || book.BookId <= 0 {
		c.Html404()
		return
	}

	uri := c.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", identify)
	code, err := qr.Encode(uri, qr.L, qr.Unicode)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.Html404()
		return
	}
	code, err = barcode.Scale(code, 150, 150)

	if err != nil {
		mus.Logger.Error(err.Error())
		c.Html404()
	}
	c.Header("Content-Type", "image/png")

	//imgpath := filepath.Join("cache","qrcode",identify + ".png")

	err = png.Encode(c.Context.Writer, code)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.Html404()
		return
	}
}

//项目内搜索.
func Search(c *core.Context) {
	identify := c.Param(":key")
	token := c.GetString("token")
	keyword := strings.TrimSpace(c.GetString("keyword"))

	if identify == "" {
		c.JSONErrStr(6001, "参数错误")
	}
	if !dao.Global.IsEnableAnonymous() && c.Member() == nil {
		c.Redirect(302, "/login")
		return
	}
	bookResult := isReadable(c, identify, token)

	client := dao.NewElasticSearchClient()
	if client.On { // 全文搜索
		result, err := client.Search(keyword, 1, 10000, true, bookResult.BookId)
		if err != nil {
			mus.Logger.Error(err.Error())
			c.JSONErrStr(6002, "搜索结果错误")
			return
		}

		var ids []int
		for _, item := range result.Hits.Hits {
			ids = append(ids, item.Source.Id)
		}
		docs, err := dao.DocumentSearchResult.GetDocsById(ids, true)
		if err != nil {
			mus.Logger.Error(err.Error())
			return
		}

		// 如果全文搜索查询不到结果，用 MySQL like 再查询一次
		if len(docs) == 0 {
			if docsMySQL, _, err := dao.DocumentSearchResult.SearchDocument(keyword, bookResult.BookId, 1, 10000); err != nil {
				mus.Logger.Error(err.Error())
				c.JSONErrStr(6002, "搜索结果错误")
				return
			} else {
				c.JSONOK(client.SegWords(keyword), docsMySQL)
			}
		} else {
			c.JSONOK(client.SegWords(keyword), docs)
		}

	} else {
		docs, _, err := dao.DocumentSearchResult.SearchDocument(keyword, bookResult.BookId, 1, 10000)
		if err != nil {
			mus.Logger.Error(err.Error())
			c.JSONErrStr(6002, "搜索结果错误")
			return
		}
		c.JSONOK(keyword, docs)
	}
}

//文档历史列表.
func History(c *core.Context) {
	identify := c.Query("identify")
	docId, _ := strconv.Atoi(c.Query("doc_id"))
	pageIndex, _ := strconv.Atoi(c.Query("page"))
	if pageIndex == 0 {
		pageIndex = 1
	}
	bookId := 0
	//如果是超级管理员则忽略权限判断
	if c.Member().IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.Tpl().Data["ErrorMessage"] = "项目不存在或权限不足"
			c.Html("document/history")
			return
		}
		bookId = book.BookId
		c.Tpl().Data["Model"] = book
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, c.Member().MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			c.Tpl().Data["ErrorMessage"] = "项目不存在或权限不足"
			c.Html("document/history")

			return
		}
		bookId = bookResult.BookId
		c.Tpl().Data["Model"] = bookResult
	}

	if docId <= 0 {
		c.Tpl().Data["ErrorMessage"] = "参数错误"
		c.Html("document/history")
		return
	}

	doc, err := dao.Document.Find(docId)
	if err != nil {
		mus.Logger.Error(err.Error())
		c.Tpl().Data["ErrorMessage"] = "获取历史失败"
		c.Html("document/history")
		return
	}
	//如果文档所属项目错误
	if doc.BookId != bookId {
		c.Tpl().Data["ErrorMessage"] = "参数错误"
		c.Html("document/history")
		return
	}

	// todo fix

	histories, totalCount, err := dao.DocumentHistory.FindToPager(docId, pageIndex, conf.PageSize)
	if err != nil {
		c.Tpl().Data["ErrorMessage"] = "获取历史失败"
		c.Html("document/history")
		return
	}

	c.Tpl().Data["List"] = histories
	c.Tpl().Data["PageHtml"] = ""
	c.Tpl().Data["Document"] = doc

	if totalCount > 0 {
		html := utils.GetPagerHtml(c.Context.Request.RequestURI, pageIndex, conf.PageSize, totalCount)
		c.Tpl().Data["PageHtml"] = html
	}
	c.Html("document/history")
}

func DeleteHistory(c *core.Context) {
	identify := c.GetString("identify")
	docId := c.GetInt("doc_id")
	historyId := c.GetInt("history_id")

	if historyId <= 0 {
		c.JSONErrStr(6001, "参数错误")
		return
	}
	bookId := 0
	//如果是超级管理员则忽略权限判断
	if c.Member().IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
		bookId = book.BookId
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, c.Member().MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		c.JSONErrStr(6001, "参数错误")
		return
	}

	doc, err := dao.Document.Find(docId)
	if err != nil {
		c.JSONErrStr(6001, "获取历史失败")
		return
	}

	//如果文档所属项目错误
	if doc.BookId != bookId {
		c.JSONErrStr(6001, "参数错误")
		return
	}

	//err = mysql.NewDocumentHistory().Delete(history_id, doc_id)
	// todo fix
	//err = mysql.NewDocumentHistory().DeleteByHistoryId(historyId)
	//if err != nil {
	//	mus.Logger.Error(err)
	//	c.JSONErrStr(6002, "删除失败")
	//}
	c.JSONOK()
}

func RestoreHistory(c *core.Context) {

	identify := c.GetString("identify")
	docId := c.GetInt("doc_id")

	historyId := c.GetInt("history_id")
	if historyId <= 0 {
		c.JSONErrStr(6001, "参数错误")
		return
	}

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if c.Member().IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
		bookId = book.BookId
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, c.Member().MemberId)
		if err != nil || bookResult.RoleId == conf.BookObserver {
			c.JSONErrStr(6002, "项目不存在或权限不足")
			return
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		c.JSONErrStr(6001, "参数错误")
		return
	}

	doc, err := dao.Document.Find(docId)

	if err != nil {
		c.JSONErrStr(6001, "获取历史失败")
		return
	}
	//如果文档所属项目错误
	if doc.BookId != bookId {
		c.JSONErrStr(6001, "参数错误")
	}

	// todo fix
	//err = mysql.NewDocumentHistory().Restore(historyId, docId, c.Member().MemberId)
	//if err != nil {
	//	mus.Logger.Error(err)
	//	c.JSONErrStr(6002, "删除失败")
	//}
	c.JSONOK()
}

func CompareHtml(c *core.Context) {
	historyId, _ := strconv.Atoi(c.Param(":id"))
	identify := c.Param(":key")

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if c.Member().IsAdministrator() {
		book, err := dao.Book.FindByFieldFirst("identify", identify)
		if err != nil {
			mus.Logger.Error("CompareHtml error ", zap.Error(err))
			c.Html404()
			return
		}
		bookId = book.BookId
		c.Tpl().Data["Model"] = book
	} else {
		bookResult, err := dao.Book.ResultFindByIdentify(identify, c.Member().MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			c.Html404()
			return
		}
		bookId = bookResult.BookId
		c.Tpl().Data["Model"] = bookResult
	}

	if historyId <= 0 {
		c.JSONErrStr(60002, "参数错误")
		return
	}

	// todo fix
	//history, err := mysql.NewDocumentHistory().Find(historyId)
	//if err != nil {
	//	mus.Logger.Error("DocumentController.Compare => ", err)
	//	this.ShowErrorPage(60003, err.Error())
	//}
	//doc, err := dao.Document.Find(history.DocumentId)
	//
	//if doc.BookId != bookId {
	//	this.ShowErrorPage(60002, "参数错误")
	//}
	//vc := mysql.NewVersionControl(doc.DocumentId, history.Version)
	c.Tpl().Data["HistoryId"] = historyId
	//c.Tpl().Data["DocumentId"] = doc.DocumentId
	//ModelStore := new(mysql.DocumentStore)
	//c.Tpl().Data["HistoryContent"] = vc.GetVersionContent(false)
	fmt.Println("bookId------>", bookId)
	//c.Tpl().Data["Content"] = ModelStore.GetFiledById(doc.DocumentId, "markdown")
	c.Html("document/compare")
}

func enableDocumentHistory() int {
	option, err := dao.Global.FindByKey("ENABLE_DOCUMENT_HISTORY")
	if err != nil {
		return 0
	}
	verNum, _ := strconv.Atoi(option.OptionValue)
	return verNum
}
