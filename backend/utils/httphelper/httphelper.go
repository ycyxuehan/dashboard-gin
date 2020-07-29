package httphelper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/gin-gonic/gin"
)

const SUCCESS = 0

//ResponseMessage 返回信息
type ResponseMessage struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}

//RestfullResponse 返回restful风格
func RestfullResponse(c *gin.Context, code int, v interface{}){
	if code == http.StatusOK {
		code = 0
	}
	resp := ResponseMessage{
		Code: code,
		Data: v,
	}
	if code != 0 && code != 200 {
		resp.Message = fmt.Sprintf("%v", v)
	}
	if code == 0 || code >=1000{
		c.JSON(http.StatusOK, &resp)
	}else {
		c.String(code, "%v", v)
	}
}

//ReadRequestBody 读取请求数据
func ReadRequestBody(c *gin.Context, v interface{})error{
	body := c.Request.Body
	defer body.Close()
	data ,err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
