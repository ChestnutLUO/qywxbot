curl 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=d0b52f14-0ca3-4b44-94bb-e901d034045b' \
   -H 'Content-Type: application/json' \
   -d '
   {
    	"msgtype": "text",
    	"text": {
        	"content": "hello world"
    	}
   }'
