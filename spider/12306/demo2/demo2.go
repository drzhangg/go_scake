package demo2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// 购买火车票需要的Restful API
//cmd中设置代理   set HTTP_proxy=http://xxx.xxx.xxx.xxx:xxxx
//https://kyfw.12306.cn/otn/passengers/query
//pageIndex: 1
//pageSize: 10
//车站Code ，当变化时重新查询车站列表信息
var stationCodeAPI = "https://kyfw.12306.cn/otn/resources/js/framework/station_name.js"

//查票价
var queryPriceAPI = "https://kyfw.12306.cn/otn/leftTicket/queryTicketPrice"

// 获取乘车人信息
var querypassagerAPI = "https://kyfw.12306.cn/otn/passengers/query"

// 获取登录时验证码图片
var captchaImageAPI = "https://kyfw.12306.cn/passport/captcha/captcha-image?login_site=E&module=login&rand=sjrand&0.6758635422370265"

//验证码验证链接
var captchaCheckAPI = "https://kyfw.12306.cn/passport/captcha/captcha-check"

//根据用户名密码进行登录，共需要如下3个API
//var loginAPI = "https://kyfw.12306.cn/passport/web/login"
var loginAPI = "https://kyfw.12306.cn/otn/resources/login"
var uamtkAPI = "https://kyfw.12306.cn/passport/web/auth/uamtk"
var uamauthclientAPI = "https://kyfw.12306.cn/otn/uamauthclient"

//查余票  后面加上参数
//var queryTicketAPI = "https://kyfw.12306.cn/otn/leftTicket/queryZ?"
var queryTicketAPI = "https://kyfw.12306.cn/otn/leftTicket/init?linktypeid=dc&fs=%E5%B9%BF%E5%B7%9E,GZQ&ts=%E5%91%A8%E5%8F%A3,ZKN&date=2020-01-20&flag=N,N,Y"

//检查用户是否在铁总
var checkUserAPI = "https://kyfw.12306.cn/otn/login/checkUser"

// 确定订单信息
var submitOrderRequestAPI = "https://kyfw.12306.cn/otn/leftTicket/submitOrderRequest"

// initDc,获取globalRepeatSubmitToken
var initDcAPI = "https://kyfw.12306.cn/otn/confirmPassenger/initDc"

//获取曾经用户列表
var getPassengerDTOsAPI = "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs"

//检查订单信息
var checkOrderInfoAPI = "https://kyfw.12306.cn/otn/confirmPassenger/checkOrderInfo"

//获取队列查询
var getQueueCountAPI = "https://kyfw.12306.cn/otn/confirmPassenger/getQueueCount"

// 确认队列 正式开始进行购票
var confirmSingleForQueueAPI = "https://kyfw.12306.cn/otn/confirmPassenger/confirmSingleForQueue"

//循环获取orderid明确是否成功    获得取票时的取票码   orderid
var queryOrderWaitTimeAPI = "https://kyfw.12306.cn/otn/confirmPassenger/queryOrderWaitTime"

//最终订单确认 信息
var resultOrderForDcQueueAPI = "https://kyfw.12306.cn/otn/confirmPassenger/resultOrderForDcQueue"

//明确是否有遗留订单    https://kyfw.12306.cn/otn/queryOrder/queryMyOrderNoComplete
// 参数  _json_att =""data["orderDBList"][0]["tickets"][index]中，这是一个数组，数组的长度就是下单的票数（就像上面下单数为2张，所以长度为2），
var queryMyOrderNoCompleteAPI = "https://kyfw.12306.cn/otn/queryOrder/queryMyOrderNoComplete"

// 退出登录  //
var logoutAPI = "https://kyfw.12306.cn/otn/login/loginOut"
var cancelNoCompleteMyOrderAPI = "https://kyfw.12306.cn/otn/queryOrder/cancelNoCompleteMyOrder"

/**
取消订单：  https://kyfw.12306.cn/otn/queryOrder/cancelNoCompleteMyOrder
参数：
sequence_no: EB92307479
cancel_flag: cancel_order

以C打头的车次：城际动车组列车
以D打头的车次：动车组列车
以G打头的车次：高速动车组列车
以Z打头的车次：直达特别快速旅客列车
以T打头的车次：特别快速旅客列车
以K打头的车次：快速旅客列车
以N打头的车次：管内快速旅客列车
L——临时“L”字头列车，代表临时列车 加增的，不常见应该（节假日）
A——临外临“A”字头列车 临时加增，突发情况
Y——旅游“Y”字头列车，代表旅游列车 我不懂
---------------------

12 跳转至已订得车票页面
url = 'https://kyfw.12306.cn/otn//payOrder/init?random=' + t
        res = self.session.post(url, verify=False)
resultOrderForDcQueue()

*/
//所有车次信息
//https://kyfw.12306.cn/otn/resources/js/query/train_list.js?scriptVersion=1.0

// 查询车票次数

var searchTickTimes int64 = 0

// 火车站 station 列表map
var stationMap = make(map[string]string)
var seatTypeMap = make(map[string]string)
var seatTicketMap = make(map[string]int)
var coordinates = make(map[string]string)

// 处理登录及cookie
var gCurCookies []*http.Cookie
var CurCookies []*http.Cookie
var gCurCookieJar *cookiejar.Jar

func initAll() {
	gCurCookies = nil
	CurCookies = nil
	gCurCookieJar, _ = cookiejar.New(nil)
	// 验证码坐标
	coordinates = map[string]string{
		"1": "45,45", "2": "120,45", "3": "180,45", "4": "255,45", "5": "45,120", "6": "120,120", "7": "180,120", "8": "255,120",
	}
	// 火车站列表
	var station_names = "@bjb|北京北|VAP|beijingbei|bjb|0@bjd|北京东|BOP|beijingdong|bjd|1@bji|北京|BJP|beijing|bj|2@bjn|北京南|VNP|beijingnan|bjn|3@bjx|北京西|BXP|beijingxi|bjx|4@gzn|广州南|IZQ|guangzhounan|gzn|5@cqb|重庆北|CUW|chongqingbei|cqb|6@cqi|重庆|CQW|chongqing|cq|7@cqn|重庆南|CRW|chongqingnan|cqn|8@cqx|重庆西|CXW|chongqingxi|cqx|9@gzd|广州东|GGQ|guangzhoudong|gzd|10@sha|上海|SHH|shanghai|sh|11@shn|上海南|SNH|shanghainan|shn|12@shq|上海虹桥|AOH|shanghaihongqiao|shhq|13@shx|上海西|SXH|shanghaixi|shx|14@tjb|天津北|TBP|tianjinbei|tjb|15@tji|天津|TJP|tianjin|tj|16@tjn|天津南|TIP|tianjinnan|tjn|17@tjx|天津西|TXP|tianjinxi|tjx|18@xgl|香港西九龙|XJA|hkwestkowloon|xgxjl|19@cch|长春|CCT|changchun|cc|20@ccn|长春南|CET|changchunnan|ccn|21@ccx|长春西|CRT|changchunxi|ccx|22@cdd|成都东|ICW|chengdudong|cdd|23@cdn|成都南|CNW|chengdunan|cdn|24@cdu|成都|CDW|chengdu|cd|25@csh|长沙|CSQ|changsha|cs|26@csn|长沙南|CWQ|changshanan|csn|27@dmh|大明湖|JAK|daminghu|dmh|28@fzh|福州|FZS|fuzhou|fz|29@fzn|福州南|FYS|fuzhounan|fzn|30@gya|贵阳|GIW|guiyang|gy|31@gzh|广州|GZQ|guangzhou|gz|32@gzx|广州西|GXQ|guangzhouxi|gzx|33@heb|哈尔滨|HBB|haerbin|heb|34@hed|哈尔滨东|VBB|haerbindong|hebd|35@hex|哈尔滨西|VAB|haerbinxi|hebx|36@hfe|合肥|HFH|hefei|hf|37@hfx|合肥西|HTH|hefeixi|hfx|38@hhd|呼和浩特东|NDC|huhehaotedong|hhhtd|39@hht|呼和浩特|HHC|huhehaote|hhht|40@hkd|海  口东|KEQ|haikoudong|hkd|41@hkd|海口东|HMQ|haikoudong|hkd|42@hko|海口|VUQ|haikou|hk|43@hzd|杭州东|HGH|hangzhoudong|hzd|44@hzh|杭州|HZH|hangzhou|hz|45@hzn|杭州南|XHH|hangzhounan|hzn|46@jna|济南|JNK|jinan|jn|47@jnx|济南西|JGK|jinanxi|jnx|48@kmi|昆明|KMM|kunming|km|49@kmx|昆明西|KXM|kunmingxi|kmx|50@lsa|拉萨|LSO|lasa|ls|51@lzd|兰州东|LVJ|lanzhoudong|lzd|52@lzh|兰州|LZJ|lanzhou|lz|53@lzx|兰州西|LAJ|lanzhouxi|lzx|54@nch|南昌|NCG|nanchang|nc|55@nji|南京|NJH|nanjing|nj|56@njn|南京南|NKH|nanjingnan|njn|57@nni|南宁|NNZ|nanning|nn|58@sjb|石家庄北|VVP|shijiazhuangbei|sjzb|59@sjz|石家庄|SJP|shijiazhuang|sjz|60@sya|沈阳|SYT|shenyang|sy|61@syb|沈阳北|SBT|shenyangbei|syb|62@syd|沈阳东|SDT|shenyangdong|syd|63@syn|沈阳南|SOT|shenyangnan|syn|64@tyb|太原北|TBV|taiyuanbei|tyb|65@tyd|太原东|TDV|taiyuandong|tyd|66@tyu|太原|TYV|taiyuan|ty|67@wha|武汉|WHN|wuhan|wh|68@wjx|王家营西|KNM|wangjiayingxi|wjyx|69@wlq|乌鲁木齐|WAR|wulumuqi|wlmq|70@xab|西安北|EAY|xianbei|xab|71@xan|西安|XAY|xian|xa|72@xan|西安南|CAY|xiannan|xan|73@xni|西宁|XNO|xining|xn|74@ych|银川|YIJ|yinchuan|yc|75@zzh|郑州|ZZF|zhengzhou|zz|76@aes|阿尔山|ART|aershan|aes|77@aka|安康|AKY|ankang|ak|78@aks|阿克苏|ASR|akesu|aks|79@alh|阿里河|AHX|alihe|alh|80@alk|阿拉山口|AKR|alashankou|alsk|81@api|安平|APT|anping|ap|82@aqi|安庆|AQH|anqing|aq|83@ash|安顺|ASW|anshun|as|84@ash|鞍山|AST|anshan|as|85@aya|安阳|AYF|anyang|ay|86@ban|北安|BAB|beian|ba|87@bbu|蚌埠|BBH|bengbu|bb|88@bch|白城|BCT|baicheng|bc|89@bha|北海|BHZ|beihai|bh|90@bhe|白河|BEL|baihe|bh|91@bji|白涧|BAP|baijian|bj|92@bji|宝鸡|BJY|baoji|bj|93@bji|滨江|BJB|binjiang|bj|94@bkt|博克图|BKX|boketu|bkt|95@bse|百色|BIZ|baise|bs|96@bss|白山市|HJL|baishanshi|bss|97@bta|北台|BTT|beitai|bt|98@btd|包头东|BDC|baotoudong|btd|99@bto|包头|BTC|baotou|bt|100@bts|北屯市|BXR|beitunshi|bts|101@bxi|本溪|BXT|benxi|bx|102@byb|白云鄂博|BEC|baiyunebo|byeb|103@byx|白银西|BXJ|baiyinxi|byx|104@bzh|亳州|BZH|bozhou|bz|105@cbi|赤壁|CBN|chibi|cb|106@cde|常德|VGQ|changde|cd|107@cde|承德|CDP|chengde|cd|108@cdi|长甸|CDT|changdian|cd|109@cfe|赤峰|CFD|chifeng|cf|110@cli|茶陵|CDG|chaling|cl|111@cna|苍南|CEH|cangnan|cn|112@cpi|昌平|CPP|changping|cp|113@cre|崇仁|CRG|chongren|cr|114@ctu|昌图|CTT|changtu|ct|115@ctz|长汀镇|CDB|changtingzhen|ctz|116@cxi|曹县|CXK|caoxian|cx|117@cxn|楚雄南|COM|chuxiongnan|cxn|118@cxt|陈相屯|CXT|chenxiangtun|cxt|119@czb|长治北|CBF|changzhibei|czb|120@czh|池州|IYH|chizhou|cz|121@czh|长征|CZJ|changzheng|cz|122@czh|常州|CZH|changzhou|cz|123@czh|郴州|CZQ|chenzhou|cz|124@czh|长治|CZF|changzhi|cz|125@czh|沧州|COP|cangzhou|cz|126@czu|崇左|CZZ|chongzuo|cz|127@dab|大安北|RNT|daanbei|dab|128@dch|大成|DCT|dacheng|dc|129@ddo|丹东|DUT|dandong|dd|130@dfh|东方红|DFB|dongfanghong|dfh|131@dgd|东莞东|DMQ|dongguandong|dgd|132@dhs|大虎山|DHD|dahushan|dhs|133@dhu|敦化|DHL|dunhua|dh|134@dhu|敦煌|DHJ|dunhuang|dh|135@dhu|德惠|DHT|dehui|dh|136@djc|东京城|DJB|dongjingcheng|djc|137@dji|大涧|DFP|dajian|dj|138@djy|都江堰|DDW|dujiangyan|djy|139@dlb|大连北|DFT|dalianbei|dlb|140@dli|大理|DKM|dali|dl|141@dli|大连|DLT|dalian|dl|142@dna|定南|DNG|dingnan|dn|143@dqi|大庆|DZX|daqing|dq|144@dsh|东胜|DOC|dongsheng|ds|145@dsq|大石桥|DQT|dashiqiao|dsq|146@dto|大同|DTV|datong|dt|147@dyi|东营|DPK|dongying|dy|148@dys|大杨树|DUX|dayangshu|dys|149@dyu|都匀|RYW|duyun|dy|150@dzh|邓州|DOF|dengzhou|dz|151@dzh|达州|RXW|dazhou|dz|152@dzh|德州|DZP|dezhou|dz|153@ejn|额济纳|EJC|ejina|ejn|154@eli|二连|RLC|erlian|el|155@esh|恩施|ESN|enshi|es|156@fdi|福鼎|FES|fuding|fd|157@fhc|凤凰机场|FJQ|fenghuangjichang|fhjc|158@fld|风陵渡|FLV|fenglingdu|fld|159@fli|涪陵|FLW|fuling|fl|160@flj|富拉尔基|FRX|fulaerji|flej|161@fsb|抚顺北|FET|fushunbei|fsb|162@fsh|佛山|FSQ|foshan|fs|163@fxn|阜新南|FXD|fuxinnan|fxn|164@fya|阜阳|FYH|fuyang|fy|165@gem|格尔木|GRO|geermu|gem|166@gha|广汉|GHW|guanghan|gh|167@gji|古交|GJV|gujiao|gj|168@glb|桂林北|GBZ|guilinbei|glb|169@gli|古莲|GRX|gulian|gl|170@gli|桂林|GLZ|guilin|gl|171@gsh|固始|GXN|gushi|gs|172@gsh|广水|GSN|guangshui|gs|173@gta|干塘|GNJ|gantang|gt|174@gyu|广元|GYW|guangyuan|gy|175@gzb|广州北|GBQ|guangzhoubei|gzb|176@gzh|赣州|GZG|ganzhou|gz|177@gzl|公主岭|GLT|gongzhuling|gzl|178@gzn|公主岭南|GBT|gongzhulingnan|gzln|179@han|淮安|AUH|huaian|ha|180@hbe|淮北|HRH|huaibei|hb|181@hbe|鹤北|HMB|hebei|hb|182@hbi|淮滨|HVN|huaibin|hb|183@hbi|河边|HBV|hebian|hb|184@hch|潢川|KCN|huangchuan|hc|185@hch|韩城|HCY|hancheng|hc|186@hda|邯郸|HDP|handan|hd|187@hdz|横道河子|HDB|hengdaohezi|hdhz|188@hga|鹤岗|HGB|hegang|hg|189@hgt|皇姑屯|HTT|huanggutun|hgt|190@hgu|红果|HEM|hongguo|hg|191@hhe|黑河|HJB|heihe|hh|192@hhu|怀化|HHQ|huaihua|hh|193@hko|汉口|HKN|hankou|hk|194@hld|葫芦岛|HLD|huludao|hld|195@hle|海拉尔|HRX|hailaer|hle|196@hll|霍林郭勒|HWD|huolinguole|hlgl|197@hlu|海伦|HLB|hailun|hl|198@hma|侯马|HMV|houma|hm|199@hmi|哈密|HMR|hami|hm|200@hna|淮南|HAH|huainan|hn|201@hna|桦南|HNB|huanan|hn|202@hnx|海宁西|EUH|hainingxi|hnx|203@hqi|鹤庆|HQM|heqing|hq|204@hrb|怀柔北|HBP|huairoubei|hrb|205@hro|怀柔|HRP|huairou|hr|206@hsd|黄石东|OSN|huangshidong|hsd|207@hsh|华山|HSY|huashan|hs|208@hsh|黄山|HKH|huangshan|hs|209@hsh|黄石|HSN|huangshi|hs|210@hsh|衡水|HSP|hengshui|hs|211@hya|衡阳|HYQ|hengyang|hy|212@hze|菏泽|HIK|heze|hz|213@hzh|贺州|HXZ|hezhou|hz|214@hzh|汉中|HOY|hanzhong|hz|215@hzh|惠州|HCQ|huizhou|hz|216@jan|吉安|VAG|jian|ja|217@jan|集安|JAL|jian|ja|218@jbc|江边村|JBG|jiangbiancun|jbc|219@jch|晋城|JCF|jincheng|jc|220@jcj|金城江|JJZ|jinchengjiang|jcj|221@jdz|景德镇|JCG|jingdezhen|jdz|222@jfe|嘉峰|JFF|jiafeng|jf|223@jgq|加格达奇|JGX|jiagedaqi|jgdq|224@jgs|井冈山|JGG|jinggangshan|jgs|225@jhe|蛟河|JHL|jiaohe|jh|226@jhn|金华南|RNH|jinhuanan|jhn|227@jhu|金华|JBH|jinhua|jh|228@jji|九江|JJG|jiujiang|jj|229@jli|吉林|JLL|jilin|jl|230@jme|荆门|JMN|jingmen|jm|231@jms|佳木斯|JMB|jiamusi|jms|232@jni|济宁|JIK|jining|jn|233@jnn|集宁南|JAC|jiningnan|jnn|234@jqu|酒泉|JQJ|jiuquan|jq|235@jsh|江山|JUH|jiangshan|js|236@jsh|吉首|JIQ|jishou|js|237@jta|九台|JTL|jiutai|jt|238@jts|镜铁山|JVJ|jingtieshan|jts|239@jxi|鸡西|JXB|jixi|jx|240@jxx|绩溪县|JRH|jixixian|jxx|241@jyg|嘉峪关|JGJ|jiayuguan|jyg|242@jyo|江油|JFW|jiangyou|jy|243@jzb|蓟州北|JKP|jizhoubei|jzb|244@jzh|金州|JZT|jinzhou|jz|245@jzh|锦州|JZD|jinzhou|jz|246@kel|库尔勒|KLR|kuerle|kel|247@kfe|开封|KFF|kaifeng|kf|248@kla|岢岚|KLV|kelan|kl|249@kli|凯里|KLW|kaili|kl|250@ksh|喀什|KSR|kashi|ks|251@ksn|昆山南|KNH|kunshannan|ksn|252@ktu|奎屯|KTR|kuitun|kt|253@kyu|开原|KYT|kaiyuan|ky|254@lan|六安|UAH|luan|la|255@lba|灵宝|LBF|lingbao|lb|256@lcg|芦潮港|UCH|luchaogang|lcg|257@lch|隆昌|LCW|longchang|lc|258@lch|陆川|LKZ|luchuan|lc|259@lch|利川|LCN|lichuan|lc|260@lch|临川|LCG|linchuan|lc|261@lch|潞城|UTP|lucheng|lc|262@lda|鹿道|LDL|ludao|ld|263@ldi|娄底|LDQ|loudi|ld|264@lfe|临汾|LFV|linfen|lf|265@lgz|良各庄|LGP|lianggezhuang|lgz|266@lhe|临河|LHC|linhe|lh|267@lhe|漯河|LON|luohe|lh|268@lhu|绿化|LWJ|lvhua|lh|269@lhu|隆化|UHP|longhua|lh|270@lji|丽江|LHM|lijiang|lj|271@lji|临江|LQL|linjiang|lj|272@lji|龙井|LJL|longjing|lj|273@lli|吕梁|LHV|lvliang|ll|274@lli|醴陵|LLG|liling|ll|275@lln|柳林南|LKV|liulinnan|lln|276@lpi|滦平|UPP|luanping|lp|277@lps|六盘水|UMW|liupanshui|lps|278@lqi|灵丘|LVV|lingqiu|lq|279@lsh|旅顺|LST|lvshun|ls|280@lxi|兰溪|LWH|lanxi|lx|281@lxi|陇西|LXJ|longxi|lx|282@lxi|澧县|LEQ|lixian|lx|283@lxi|临西|UEP|linxi|lx|284@lya|龙岩|LYS|longyan|ly|285@lya|耒阳|LYQ|leiyang|ly|286@lya|洛阳|LYF|luoyang|ly|287@lyd|连云港东|UKH|lianyungangdong|lygd|288@lyd|洛阳东|LDF|luoyangdong|lyd|289@lyi|临沂|LVK|linyi|ly|290@lym|洛阳龙门|LLF|luoyanglongmen|lylm|291@lyu|柳园|DHR|liuyuan|ly|292@lyu|凌源|LYD|lingyuan|ly|293@lyu|辽源|LYL|liaoyuan|ly|294@lzh|立志|LZX|lizhi|lz|295@lzh|柳州|LZZ|liuzhou|lz|296@lzh|辽中|LZD|liaozhong|lz|297@mch|麻城|MCN|macheng|mc|298@mdh|免渡河|MDX|mianduhe|mdh|299@mdj|牡丹江|MDB|mudanjiang|mdj|300@meg|莫尔道嘎|MRX|moerdaoga|medg|301@mgu|明光|MGH|mingguang|mg|302@mgu|满归|MHX|mangui|mg|303@mhe|漠河|MVX|mohe|mh|304@mmi|茂名|MDQ|maoming|mm|305@mmx|茂名西|MMZ|maomingxi|mmx|306@msh|密山|MSB|mishan|ms|307@msj|马三家|MJT|masanjia|msj|308@mwe|麻尾|VAW|mawei|mw|309@mya|绵阳|MYW|mianyang|my|310@mzh|梅州|MOQ|meizhou|mz|311@mzl|满洲里|MLX|manzhouli|mzl|312@nbd|宁波东|NVH|ningbodong|nbd|313@nbo|宁波|NGH|ningbo|nb|314@nch|南岔|NCB|nancha|nc|315@nch|南充|NCW|nanchong|nc|316@nda|南丹|NDZ|nandan|nd|317@ndm|南大庙|NMP|nandamiao|ndm|318@nfe|南芬|NFT|nanfen|nf|319@nhe|讷河|NHX|nehe|nh|320@nji|嫩江|NGX|nenjiang|nj|321@nji|内江|NJW|neijiang|nj|322@npi|南平|NPS|nanping|np|323@nto|南通|NUH|nantong|nt|324@nya|南阳|NFF|nanyang|ny|325@nzs|碾子山|NZX|nianzishan|nzs|326@pds|平顶山|PEN|pingdingshan|pds|327@pji|盘锦|PVD|panjin|pj|328@pli|平凉|PIJ|pingliang|pl|329@pln|平凉南|POJ|pingliangnan|pln|330@pqu|平泉|PQP|pingquan|pq|331@psh|坪石|PSQ|pingshi|ps|332@pxi|萍乡|PXG|pingxiang|px|333@pxi|凭祥|PXZ|pingxiang|px|334@pxx|郫县西|PCW|pixianxi|pxx|335@pzh|攀枝花|PRW|panzhihua|pzh|336@qch|蕲春|QRN|qichun|qc|337@qcs|青城山|QSW|qingchengshan|qcs|338@qda|青岛|QDK|qingdao|qd|339@qhc|清河城|QYP|qinghecheng|qhc|340@qji|曲靖|QJM|qujing|qj|341@qji|黔江|QNW|qianjiang|qj|342@qjz|前进镇|QEB|qianjinzhen|qjz|343@qqe|齐齐哈尔|QHX|qiqihaer|qqhe|344@qth|七台河|QTB|qitaihe|qth|345@qxi|沁县|QVV|qinxian|qx|346@qzd|泉州东|QRS|quanzhoudong|qzd|347@qzh|泉州|QYS|quanzhou|qz|348@qzh|衢州|QEH|quzhou|qz|349@ran|融安|RAZ|rongan|ra|350@rjg|汝箕沟|RQJ|rujigou|rjg|351@rji|瑞金|RJG|ruijin|rj|352@rzh|日照|RZK|rizhao|rz|353@scp|双城堡|SCB|shuangchengpu|scp|354@sfh|绥芬河|SFB|suifenhe|sfh|355@sgd|韶关东|SGQ|shaoguandong|sgd|356@shg|山海关|SHD|shanhaiguan|shg|357@shu|绥化|SHB|suihua|sh|358@sjf|三间房|SFX|sanjianfang|sjf|359@sjt|苏家屯|SXT|sujiatun|sjt|360@sla|舒兰|SLL|shulan|sl|361@smn|神木南|OMY|shenmunan|smn|362@smx|三门峡|SMF|sanmenxia|smx|363@sna|商南|ONY|shangnan|sn|364@sni|遂宁|NIW|suining|sn|365@spi|四平|SPT|siping|sp|366@sqi|商丘|SQF|shangqiu|sq|367@sra|上饶|SRG|shangrao|sr|368@ssh|韶山|SSQ|shaoshan|ss|369@sso|宿松|OAH|susong|ss|370@sto|汕头|OTQ|shantou|st|371@swu|邵武|SWS|shaowu|sw|372@sxi|涉县|OEP|shexian|sx|373@sya|三亚|SEQ|sanya|sy|374@sya|三  亚|JUQ|sanya|sya|375@sya|邵阳|SYQ|shaoyang|sy|376@sya|十堰|SNN|shiyan|sy|377@syq|三元区|SMS|sanyuanqu|syq|378@sys|双鸭山|SSB|shuangyashan|sys|379@syu|松原|VYT|songyuan|sy|380@szh|苏州|SZH|suzhou|sz|381@szh|深圳|SZQ|shenzhen|sz|382@szh|宿州|OXH|suzhou|sz|383@szh|随州|SZN|suizhou|sz|384@szh|朔州|SUV|shuozhou|sz|385@szx|深圳西|OSQ|shenzhenxi|szx|386@tba|塘豹|TBQ|tangbao|tb|387@teq|塔尔气|TVX|taerqi|teq|388@tgu|潼关|TGY|tongguan|tg|389@tgu|塘沽|TGP|tanggu|tg|390@the|塔河|TXX|tahe|th|391@thu|通化|THL|tonghua|th|392@tla|泰来|TLX|tailai|tl|393@tlf|吐鲁番|TFR|tulufan|tlf|394@tli|通辽|TLD|tongliao|tl|395@tli|铁岭|TLT|tieling|tl|396@tlz|陶赖昭|TPT|taolaizhao|tlz|397@tme|图们|TML|tumen|tm|398@tre|铜仁|RDQ|tongren|tr|399@tsb|唐山北|FUP|tangshanbei|tsb|400@tsf|田师府|TFT|tianshifu|tsf|401@tsh|泰山|TAK|taishan|ts|402@tsh|唐山|TSP|tangshan|ts|403@tsh|天水|TSJ|tianshui|ts|404@typ|通远堡|TYT|tongyuanpu|typ|405@tys|太阳升|TQT|taiyangsheng|tys|406@tzh|泰州|UTH|taizhou|tz|407@tzi|桐梓|TZW|tongzi|tz|408@tzx|通州西|TAP|tongzhouxi|tzx|409@wch|五常|WCB|wuchang|wc|410@wch|武昌|WCN|wuchang|wc|411@wfd|瓦房店|WDT|wafangdian|wfd|412@wha|威海|WKK|weihai|wh|413@whu|芜湖|WHH|wuhu|wh|414@whx|乌海西|WXC|wuhaixi|whx|415@wjt|吴家屯|WJT|wujiatun|wjt|416@wln|乌鲁木齐南|WMR|wulumuqinan|wlmqn|417@wlo|武隆|WLW|wulong|wl|418@wlt|乌兰浩特|WWT|wulanhaote|wlht|419@wna|渭南|WNY|weinan|wn|420@wsh|威舍|WSM|weishe|ws|421@wts|歪头山|WIT|waitoushan|wts|422@wwe|武威|WUJ|wuwei|ww|423@wwn|武威南|WWJ|wuweinan|wwn|424@wxi|无锡|WXH|wuxi|wx|425@wxi|乌西|WXR|wuxi|wx|426@wyl|乌伊岭|WPB|wuyiling|wyl|427@wys|武夷山|WAS|wuyishan|wys|428@wyu|万源|WYY|wanyuan|wy|429@wzh|万州|WYW|wanzhou|wz|430@wzh|梧州|WZZ|wuzhou|wz|431@wzh|温州|RZH|wenzhou|wz|432@wzn|温州南|VRH|wenzhounan|wzn|433@xch|西昌|ECW|xichang|xc|434@xch|许昌|XCF|xuchang|xc|435@xcn|西昌南|ENW|xichangnan|xcn|436@xlt|锡林浩特|XTC|xilinhaote|xlht|437@xmb|厦门北|XKS|xiamenbei|xmb|438@xme|厦门|XMS|xiamen|xm|439@xmq|厦门高崎|XBS|xiamengaoqi|xmgq|440@xwe|宣威|XWM|xuanwei|xw|441@xxi|新乡|XXF|xinxiang|xx|442@xya|信阳|XUN|xinyang|xy|443@xya|咸阳|XYY|xianyang|xy|444@xya|襄阳|XFN|xiangyang|xy|445@xyc|熊岳城|XYT|xiongyuecheng|xyc|446@xyu|新余|XUG|xinyu|xy|447@xzh|徐州|XCH|xuzhou|xz|448@yan|延安|YWY|yanan|ya|449@ybi|宜宾|YBW|yibin|yb|450@ybn|亚布力南|YWB|yabulinan|ybln|451@ybs|叶柏寿|YBD|yebaishou|ybs|452@ycd|宜昌东|HAN|yichangdong|ycd|453@ych|永川|YCW|yongchuan|yc|454@ych|盐城|AFH|yancheng|yc|455@ych|宜昌|YCN|yichang|yc|456@ych|运城|YNV|yuncheng|yc|457@ych|伊春|YCB|yichun|yc|458@yci|榆次|YCV|yuci|yc|459@ycu|杨村|YBP|yangcun|yc|460@ycx|宜春西|YCG|yichunxi|ycx|461@yes|伊尔施|YET|yiershi|yes|462@yga|燕岗|YGW|yangang|yg|463@yji|永济|YIV|yongji|yj|464@yji|延吉|YJL|yanji|yj|465@yko|营口|YKT|yingkou|yk|466@yks|牙克石|YKX|yakeshi|yks|467@yli|阎良|YNY|yanliang|yl|468@yli|玉林|YLZ|yulin|yl|469@yli|榆林|ALY|yulin|yl|470@ylw|亚龙湾|TWQ|yalongwan|ylw|471@ymp|一面坡|YPB|yimianpo|ymp|472@yni|伊宁|YMR|yining|yn|473@ypg|阳平关|YAY|yangpingguan|ypg|474@ypi|玉屏|YZW|yuping|yp|475@ypi|原平|YPV|yuanping|yp|476@yqi|延庆|YNP|yanqing|yq|477@yqq|阳泉曲|YYV|yangquanqu|yqq|478@yqu|玉泉|YQB|yuquan|yq|479@yqu|阳泉|AQP|yangquan|yq|480@ysh|营山|NUW|yingshan|ys|481@ysh|玉山|YNG|yushan|ys|482@ysh|燕山|AOP|yanshan|ys|483@ysh|榆树|YRT|yushu|ys|484@yta|鹰潭|YTG|yingtan|yt|485@yta|烟台|YAK|yantai|yt|486@yth|伊图里河|YEX|yitulihe|ytlh|487@ytx|玉田县|ATP|yutianxian|ytx|488@ywu|义乌|YWH|yiwu|yw|489@yxi|阳新|YON|yangxin|yx|490@yxi|义县|YXD|yixian|yx|491@yya|益阳|AEQ|yiyang|yy|492@yya|岳阳|YYQ|yueyang|yy|493@yzh|崖州|YUQ|yazhou|yz|494@yzh|永州|AOQ|yongzhou|yz|495@yzh|扬州|YLH|yangzhou|yz|496@zbo|淄博|ZBK|zibo|zb|497@zcd|镇城底|ZDV|zhenchengdi|zcd|498@zgo|自贡|ZGW|zigong|zg|499@zha|珠海|ZHQ|zhuhai|zh|500@zhb|珠海北|ZIQ|zhuhaibei|zhb|501@zji|湛江|ZJZ|zhanjiang|zj|502@zji|镇江|ZJH|zhenjiang|zj|503@zjj|张家界|DIQ|zhangjiajie|zjj|504@zjk|张家口|ZKP|zhangjiakou|zjk|505@zjn|张家口南|ZMP|zhangjiakounan|zjkn|506@zko|周口|ZKN|zhoukou|zk|507@zlm|哲里木|ZLC|zhelimu|zlm|508@zlt|扎兰屯|ZTX|zhalantun|zlt|509@zmd|驻马店|ZDN|zhumadian|zmd|510@zqi|肇庆|ZVQ|zhaoqing|zq|511@zsz|周水子|ZIT|zhoushuizi|zsz|512@zto|昭通|ZDW|zhaotong|zt|513@zwe|中卫|ZWJ|zhongwei|zw|514@zya|资阳|ZYW|ziyang|zy|515@zyx|遵义西|ZIW|zunyixi|zyx|516@zzh|枣庄|ZEK|zaozhuang|zz|517@zzh|资中|ZZW|zizhong|zz|518@zzh|株洲|ZZQ|zhuzhou|zz|519@zzx|枣庄西|ZFK|zaozhuangxi|zzx|520@aax|昂昂溪|AAX|angangxi|aax|521@ach|阿城|ACB|acheng|ac|522@ada|安达|ADX|anda|ad|523@ade|安德|ARW|ande|ad|524@adi|安定|ADP|anding|ad|525@adu|安多|ADO|anduo|ad|526@agu|安广|AGT|anguang|ag|527@aha|敖汉|YED|aohan|ah|528@ahe|艾河|AHP|aihe|ah|529@ahu|安化|PKQ|anhua|ah|530@ajc|艾家村|AJJ|aijiacun|ajc|531@aji|鳌江|ARH|aojiang|aj|532@aji|安家|AJB|anjia|aj|533@aji|阿金|AJD|ajin|aj|534@aji|安靖|PYW|anjing|aj|535@akt|阿克陶|AER|aketao|akt|536@aky|安口窑|AYY|ankouyao|aky|537@alg|敖力布告|ALD|aolibugao|albg|538@alo|安龙|AUZ|anlong|al|539@als|阿龙山|ASX|alongshan|als|540@alu|安陆|ALN|anlu|al|541@ame|阿木尔|JTX|amuer|ame|542@anz|阿南庄|AZM|ananzhuang|anz|543@aqx|安庆西|APH|anqingxi|aqx|544@asx|鞍山西|AXT|anshanxi|asx|545@ata|安塘|ATV|antang|at|546@atb|安亭北|ASH|antingbei|atb|547@ats|阿图什|ATR|atushi|ats|548@atu|安图|ATL|antu|at|549@axi|安溪|AXS|anxi|ax|550@bao|博鳌|BWQ|boao|ba|551@bbe|北碚|BPW|beibei|bb|552@bbg|白壁关|BGV|baibiguan|bbg|553@bbn|蚌埠南|BMH|bengbunan|bbn|554@bch|巴楚|BCR|bachu|bc|555@bch|板城|BUP|bancheng|bc|556@bdh|北戴河|BEP|beidaihe|bdh|557@bdi|保定|BDP|baoding|bd|558@bdi|宝坻|BPP|baodi|bd|559@bdl|八达岭|ILP|badaling|bdl|560@bdo|巴东|BNN|badong|bd|561@bgu|柏果|BGM|baiguo|bg|562@bha|布海|BUT|buhai|bh|563@bhd|白河东|BIY|baihedong|bhd|564@bho|贲红|BVC|benhong|bh|565@bhs|宝华山|BWH|baohuashan|bhs|566@bhx|白河县|BEY|baihexian|bhx|567@bjg|白芨沟|BJJ|baijigou|bjg|568@bjg|碧鸡关|BJM|bijiguan|bjg|569@bji|北滘|IBQ|beijiao|bj|570@bji|碧江|BLQ|bijiang|bj|571@bjp|白鸡坡|BBM|baijipo|bjp|572@bjs|笔架山|BSB|bijiashan|bjs|573@bjt|八角台|BTD|bajiaotai|bjt|574@bka|保康|BKD|baokang|bk|575@bkp|白奎堡|BKB|baikuipu|bkp|576@bla|白狼|BAT|bailang|bl|577@bla|百浪|BRZ|bailang|bl|578@ble|博乐|BOR|bole|bl|579@blg|宝拉格|BQC|baolage|blg|580@bli|巴林|BLX|balin|bl|581@bli|宝林|BNB|baolin|bl|582@bli|北流|BOZ|beiliu|bl|583@bli|勃利|BLB|boli|bl|584@blk|布列开|BLR|buliekai|blk|585@bls|宝龙山|BND|baolongshan|bls|586@blx|百里峡|AAP|bailixia|blx|587@bmc|八面城|BMD|bamiancheng|bmc|588@bmq|班猫箐|BNM|banmaoqing|bmq|589@bmt|八面通|BMB|bamiantong|bmt|590@bmz|北马圈子|BRP|beimaquanzi|bmqz|591@bpn|北票南|RPD|beipiaonan|bpn|592@bqi|白旗|BQP|baiqi|bq|593@bql|宝泉岭|BQB|baoquanling|bql|594@bqu|白泉|BQL|baiquan|bq|595@bsh|巴山|BAY|bashan|bs|596@bsj|白水江|BSY|baishuijiang|bsj|597@bsp|白沙坡|BPM|baishapo|bsp|598@bss|白石山|BAL|baishishan|bss|599@bsz|白水镇|BUM|baishuizhen|bsz|600@btd|包头 东|FDC|baotoudong|btd|601@bti|坂田|BTQ|bantian|bt|602@bto|泊头|BZP|botou|bt|603@btu|北屯|BYP|beitun|bt|604@bxh|本溪湖|BHT|benxihu|bxh|605@bxi|博兴|BXK|boxing|bx|606@bxt|八仙筒|VXD|baxiantong|bxt|607@byg|白音察干|BYC|baiyinchagan|bycg|608@byh|背荫河|BYB|beiyinhe|byh|609@byi|北营|BIV|beiying|by|610@byl|巴彦高勒|BAC|bayangaole|bygl|611@byl|白音他拉|BID|baiyintala|bytl|612@byq|鲅鱼圈|BYT|bayuquan|byq|613@bys|白银市|BNJ|baiyinshi|bys|614@bys|白音胡硕|BCD|baiyinhushuo|byhs|615@bzh|巴中|IEW|bazhong|bz|616@bzh|霸州|RMP|bazhou|bz|617@bzh|北宅|BVP|beizhai|bz|618@cbb|赤壁北|CIN|chibibei|cbb|619@cbg|查布嘎|CBC|chabuga|cbg|620@cch|长城|CEJ|changcheng|cc|621@cch|长冲|CCM|changchong|cc|622@cdd|承德东|CCP|chengdedong|cdd|623@cfx|赤峰西|CID|chifengxi|cfx|624@cga|嵯岗|CAX|cuogang|cg|625@cga|柴岗|CGT|chaigang|cg|626@cge|长葛|CEF|changge|cg|627@cgp|柴沟堡|CGV|chaigoupu|cgp|628@cgu|城固|CGY|chenggu|cg|629@cgy|陈官营|CAJ|chenguanying|cgy|630@cgz|成高子|CZB|chenggaozi|cgz|631@cha|草海|WBW|caohai|ch|632@che|柴河|CHB|chaihe|ch|633@che|册亨|CHZ|ceheng|ch|634@chk|草河口|CKT|caohekou|chk|635@chk|崔黄口|CHP|cuihuangkou|chk|636@chu|巢湖|CIH|chaohu|ch|637@cjg|蔡家沟|CJT|caijiagou|cjg|638@cjh|成吉思汗|CJX|chengjisihan|cjsh|639@cji|岔江|CAM|chajiang|cj|640@cjp|蔡家坡|CJY|caijiapo|cjp|641@cle|昌乐|CLK|changle|cl|642@clg|超梁沟|CYP|chaolianggou|clg|643@cli|慈利|CUQ|cili|cl|644@cli|昌黎|CLP|changli|cl|645@clz|长岭子|CLT|changlingzi|clz|646@cmi|晨明|CMB|chenming|cm|647@cno|长农|CNJ|changnong|cn|648@cpb|昌平北|VBP|changpingbei|cpb|649@cpi|常平|DAQ|changping|cp|650@cpl|长坡岭|CPM|changpoling|cpl|651@cqi|辰清|CQB|chenqing|cq|652@csh|蔡山|CON|caishan|cs|653@csh|楚山|CSB|chushan|cs|654@csh|长寿|EFW|changshou|cs|655@csh|磁山|CSP|cishan|cs|656@csh|苍石|CST|cangshi|cs|657@csh|草市|CSL|caoshi|cs|658@csq|察素齐|CSC|chasuqi|csq|659@cst|长山屯|CVT|changshantun|cst|660@cti|长汀|CES|changting|ct|661@ctn|朝天南|CTY|chaotiannan|ctn|662@ctx|昌图西|CPT|changtuxi|ctx|663@cwa|春湾|CQQ|chunwan|cw|664@cxi|磁县|CIP|cixian|cx|665@cxi|岑溪|CNZ|cenxi|cx|666@cxi|辰溪|CXQ|chenxi|cx|667@cxi|磁西|CRP|cixi|cx|668@cxn|长兴南|CFH|changxingnan|cxn|669@cya|磁窑|CYK|ciyao|cy|670@cya|春阳|CAL|chunyang|cy|671@cya|城阳|CEK|chengyang|cy|672@cyc|创业村|CEX|chuangyecun|cyc|673@cyc|朝阳川|CYL|chaoyangchuan|cyc|674@cyd|朝阳地|CDD|chaoyangdi|cyd|675@cyn|朝阳南|CYD|chaoyangnan|cyn|676@cyu|长垣|CYF|changyuan|cy|677@cyz|朝阳镇|CZL|chaoyangzhen|cyz|678@czb|滁州北|CUH|chuzhoubei|czb|679@czb|常州北|ESH|changzhoubei|czb|680@czh|滁州|CXH|chuzhou|cz|681@czh|潮州|CKQ|chaozhou|cz|682@czh|常庄|CVK|changzhuang|cz|683@czl|曹子里|CFP|caozili|czl|684@czw|车转湾|CWM|chezhuanwan|czw|685@czx|郴州西|ICQ|chenzhouxi|czx|686@czx|沧州西|CBP|cangzhouxi|czx|687@dan|德安|DAG|dean|da|688@dan|大安|RAT|daan|da|689@dba|大坝|DBJ|daba|db|690@dba|大板|DBC|daban|db|691@dba|大巴|DBD|daba|db|692@dba|电白|NWQ|dianbai|db|693@dba|到保|RBT|daobao|db|694@dbc|达坂城|DCR|dabancheng|dbc|695@dbi|定边|DYJ|dingbian|db|696@dbj|东边井|DBB|dongbianjing|dbj|697@dbs|德伯斯|RDT|debosi|dbs|698@dcg|打柴沟|DGJ|dachaigou|dcg|699@dch|德昌|DVW|dechang|dc|700@dda|滴道|DDB|didao|dd|701@ddg|大磴沟|DKJ|dadenggou|ddg|702@ded|刀尔登|DRD|daoerdeng|ded|703@dee|得耳布尔|DRX|deerbuer|debe|704@det|杜尔伯特|TKX|duerbote|debt|705@dfa|东方|UFQ|dongfang|df|706@dfe|丹凤|DGY|danfeng|df|707@dfe|东丰|DIL|dongfeng|df|708@dge|都格|DMM|duge|dg|709@dgt|大官屯|DTT|daguantun|dgt|710@dgu|大关|RGW|daguan|dg|711@dgu|东光|DGP|dongguang|dg|712@dha|东海|DHB|donghai|dh|713@dhc|大灰厂|DHP|dahuichang|dhc|714@dhq|大红旗|DQD|dahongqi|dhq|715@dht|大禾塘|SOQ|shaodong|dh|716@dhx|德惠西|DXT|dehuixi|dhx|717@dhx|东海县|DQH|donghaixian|dhx|718@djg|达家沟|DJT|dajiagou|djg|719@dji|东津|DKB|dongjin|dj|720@dji|杜家|DJL|dujia|dj|721@dkt|大口屯|DKP|dakoutun|dkt|722@dla|东来|RVD|donglai|dl|723@dlh|德令哈|DHO|delingha|dlh|724@dlh|大陆号|DLC|daluhao|dlh|725@dli|带岭|DLB|dailing|dl|726@dli|大林|DLD|dalin|dl|727@dlq|达拉特旗|DIC|dalateqi|dltq|728@dlt|独立屯|DTX|dulitun|dlt|729@dlu|豆罗|DLV|douluo|dl|730@dlx|达拉特西|DNC|dalatexi|dltx|731@dlx|大连西|GZT|dalianxi|dlx|732@dmc|东明村|DMD|dongmingcun|dmc|733@dmh|洞庙河|DEP|dongmiaohe|dmh|734@dmx|东明县|DNF|dongmingxian|dmx|735@dni|大拟|DNZ|dani|dn|736@dpf|大平房|DPD|dapingfang|dpf|737@dps|大盘石|RPP|dapanshi|dps|738@dpu|大埔|DPI|dapu|dp|739@dpu|大堡|DVT|dapu|dp|740@dqd|大庆东|LFX|daqingdong|dqd|741@dqh|大其拉哈|DQX|daqilaha|dqlh|742@dqi|道清|DML|daoqing|dq|743@dqs|对青山|DQB|duiqingshan|dqs|744@dqx|德清西|MOH|deqingxi|dqx|745@dqx|大庆西|RHX|daqingxi|dqx|746@dsh|东升|DRQ|dongsheng|ds|747@dsh|砀山|DKH|dangshan|ds|748@dsh|独山|RWW|dushan|ds|749@dsh|登沙河|DWT|dengshahe|dsh|750@dsp|读书铺|DPM|dushupu|dsp|751@dst|大石头|DSL|dashitou|dst|752@dsx|东胜西|DYC|dongshengxi|dsx|753@dsz|大石寨|RZT|dashizhai|dsz|754@dta|东台|DBH|dongtai|dt|755@dta|定陶|DQK|dingtao|dt|756@dta|灯塔|DGT|dengta|dt|757@dtb|大田边|DBM|datianbian|dtb|758@dth|东通化|DTL|dongtonghua|dth|759@dtu|丹徒|RUH|dantu|dt|760@dtu|大屯|DNT|datun|dt|761@dwa|东湾|DRJ|dongwan|dw|762@dwk|大武口|DFJ|dawukou|dwk|763@dwp|低窝铺|DWJ|diwopu|dwp|764@dwt|大王滩|DZZ|dawangtan|dwt|765@dwz|大湾子|DFM|dawanzi|dwz|766@dxg|大兴沟|DXL|daxinggou|dxg|767@dxi|大兴|DXX|daxing|dx|768@dxi|定西|DSJ|dingxi|dx|769@dxi|甸心|DXM|dianxin|dx|770@dxi|东乡|DXG|dongxiang|dx|771@dxi|代县|DKV|daixian|dx|772@dxi|定襄|DXV|dingxiang|dx|773@dxu|东戌|RXP|dongxu|dx|774@dxz|东辛庄|DXD|dongxinzhuang|dxz|775@dya|丹阳|DYH|danyang|dy|776@dya|德阳|DYW|deyang|dy|777@dya|大雁|DYX|dayan|dy|778@dya|当阳|DYN|dangyang|dy|779@dyb|丹阳北|EXH|danyangbei|dyb|780@dyd|大英东|IAW|dayingdong|dyd|781@dyd|东淤地|DBV|dongyudi|dyd|782@dyi|大营|DYV|daying|dy|783@dyu|定远|EWH|dingyuan|dy|784@dyu|岱岳|RYV|daiyue|dy|785@dyu|大元|DYZ|dayuan|dy|786@dyz|大营镇|DJP|dayingzhen|dyz|787@dyz|大营子|DZD|dayingzi|dyz|788@dzc|大战场|DTJ|dazhanchang|dzc|789@dzd|德州东|DIP|dezhoudong|dzd|790@dzh|东至|DCH|dongzhi|dz|791@dzh|低庄|DVQ|dizhuang|dz|792@dzh|东镇|DNV|dongzhen|dz|793@dzh|道州|DFZ|daozhou|dz|794@dzh|东庄|DZV|dongzhuang|dz|795@dzh|兑镇|DWV|duizhen|dz|796@dzh|豆庄|ROP|douzhuang|dz|797@dzh|定州|DXP|dingzhou|dz|798@dzy|大竹园|DZY|dazhuyuan|dzy|799@dzz|大杖子|DAP|dazhangzi|dzz|800@dzz|豆张庄|RZP|douzhangzhuang|dzz|801@ebi|峨边|EBW|ebian|eb|802@edm|二道沟门|RDP|erdaogoumen|edgm|803@edw|二道湾|RDX|erdaowan|edw|804@ees|鄂尔多斯|EEC|eerduosi|eeds|805@elo|二龙|RLD|erlong|el|806@elt|二龙山屯|ELA|erlongshantun|elst|807@eme|峨眉|EMW|emei|em|808@emh|二密河|RML|ermihe|emh|809@epi|恩平|PXQ|enping|ep|810@eyi|二营|RYJ|erying|ey|811@ezh|鄂州|ECN|ezhou|ez|812@fan|福安|FAS|fuan|fa|813@fch|丰城|FCG|fengcheng|fc|814@fcn|丰城南|FNG|fengchengnan|fcn|815@fdo|肥东|FIH|feidong|fd|816@fer|发耳|FEM|faer|fe|817@fha|富海|FHX|fuhai|fh|818@fha|福海|FHR|fuhai|fh|819@fhc|凤凰城|FHT|fenghuangcheng|fhc|820@fhe|汾河|FEV|fenhe|fh|821@fhu|奉化|FHH|fenghua|fh|822@fji|富锦|FIB|fujin|fj|823@fjt|范家屯|FTT|fanjiatun|fjt|824@flq|福利区|FLJ|fuliqu|flq|825@flt|福利屯|FTB|fulitun|flt|826@flz|丰乐镇|FZB|fenglezhen|flz|827@fna|阜南|FNH|funan|fn|828@fni|阜宁|AKH|funing|fn|829@fni|抚宁|FNP|funing|fn|830@fqi|福清|FQS|fuqing|fq|831@fqu|福泉|VMW|fuquan|fq|832@fsc|丰水村|FSJ|fengshuicun|fsc|833@fsh|丰顺|FUQ|fengshun|fs|834@fsh|繁峙|FSV|fanshi|fs|835@fsh|抚顺|FST|fushun|fs|836@fsk|福山口|FKP|fushankou|fsk|837@fsu|扶绥|FSZ|fusui|fs|838@ftu|冯屯|FTX|fengtun|ft|839@fty|浮图峪|FYP|futuyu|fty|840@fxd|富县东|FDY|fuxiandong|fxd|841@fxi|凤县|FXY|fengxian|fx|842@fxi|富县|FEY|fuxian|fx|843@fxi|费县|FXK|feixian|fx|844@fya|凤阳|FUH|fengyang|fy|845@fya|汾阳|FAV|fenyang|fy|846@fyb|扶余北|FBT|fuyubei|fyb|847@fyi|分宜|FYG|fenyi|fy|848@fyu|富源|FYM|fuyuan|fy|849@fyu|扶余|FYT|fuyu|fy|850@fyu|富裕|FYX|fuyu|fy|851@fzb|抚州北|FBG|fuzhoubei|fzb|852@fzh|凤州|FZY|fengzhou|fz|853@fzh|丰镇|FZC|fengzhen|fz|854@fzh|范镇|VZK|fanzhen|fz|855@gan|固安|GFP|guan|ga|856@gan|广安|VJW|guangan|ga|857@gbd|高碑店|GBP|gaobeidian|gbd|858@gbz|沟帮子|GBD|goubangzi|gbz|859@gcd|甘草店|GDJ|gancaodian|gcd|860@gch|谷城|GCN|gucheng|gc|861@gch|藁城|GEP|gaocheng|gc|862@gcu|高村|GCV|gaocun|gc|863@gcz|古城镇|GZB|guchengzhen|gcz|864@gde|广德|GRH|guangde|gd|865@gdi|贵定|GTW|guiding|gd|866@gdn|贵定南|IDW|guidingnan|gdn|867@gdo|古东|GDV|gudong|gd|868@gga|贵港|GGZ|guigang|gg|869@gga|官高|GVP|guangao|gg|870@ggm|葛根庙|GGT|gegenmiao|ggm|871@ggo|干沟|GGL|gangou|gg|872@ggu|甘谷|GGJ|gangu|gg|873@ggz|高各庄|GGP|gaogezhuang|ggz|874@ghe|甘河|GAX|ganhe|gh|875@ghe|根河|GEX|genhe|gh|876@gjd|郭家店|GDT|guojiadian|gjd|877@gjz|孤家子|GKT|gujiazi|gjz|878@gla|古浪|GLJ|gulang|gl|879@gla|皋兰|GEJ|gaolan|gl|880@glf|高楼房|GFM|gaoloufang|glf|881@glh|归流河|GHT|guiliuhe|glh|882@gli|关林|GLF|guanlin|gl|883@glu|甘洛|VOW|ganluo|gl|884@glz|郭磊庄|GLP|guoleizhuang|glz|885@gmi|高密|GMK|gaomi|gm|886@gmz|公庙子|GMC|gongmiaozi|gmz|887@gnh|工农湖|GRT|gongnonghu|gnh|888@gnn|广宁寺南|GNT|guangningsinan|gnn|889@gnw|广南卫|GNM|guangnanwei|gnw|890@gpi|高平|GPF|gaoping|gp|891@gqb|甘泉北|GEY|ganquanbei|gqb|892@gqc|共青城|GAG|gongqingcheng|gqc|893@gqk|甘旗卡|GQD|ganqika|gqk|894@gqu|甘泉|GQY|ganquan|gq|895@gqz|高桥镇|GZD|gaoqiaozhen|gqz|896@gsh|灌水|GST|guanshui|gs|897@gsh|赶水|GSW|ganshui|gs|898@gsk|孤山口|GSP|gushankou|gsk|899@gso|果松|GSL|guosong|gs|900@gsz|高山子|GSD|gaoshanzi|gsz|901@gsz|嘎什甸子|GXD|gashidianzi|gsdz|902@gta|高台|GTJ|gaotai|gt|903@gta|高滩|GAY|gaotan|gt|904@gti|古田|GTS|gutian|gt|905@gti|官厅|GTP|guanting|gt|906@gtx|官厅西|KEP|guantingxi|gtx|907@gxi|贵溪|GXG|guixi|gx|908@gya|涡阳|GYH|guoyang|gy|909@gyi|巩义|GXF|gongyi|gy|910@gyi|高邑|GIP|gaoyi|gy|911@gyn|巩义南|GYF|gongyinan|gyn|912@gyn|广元南|GAW|guangyuannan|gyn|913@gyu|固原|GUJ|guyuan|gy|914@gyu|菇园|GYL|guyuan|gy|915@gyz|公营子|GYD|gongyingzi|gyz|916@gze|光泽|GZS|guangze|gz|917@gzh|古镇|GNQ|guzhen|gz|918@gzh|固镇|GEH|guzhen|gz|919@gzh|虢镇|GZY|guozhen|gz|920@gzh|瓜州|GZJ|guazhou|gz|921@gzh|高州|GSQ|gaozhou|gz|922@gzh|盖州|GXT|gaizhou|gz|923@gzj|官字井|GOT|guanzijing|gzj|924@gzs|冠豸山|GSS|guanzhaishan|gzs|925@gzx|盖州西|GAT|gaizhouxi|gzx|926@han|海安|HIH|haian|ha|927@han|淮安南|AMH|huaiannan|han|928@han|红安|HWN|hongan|ha|929@hax|红安西|VXN|honganxi|hax|930@hba|黄柏|HBL|huangbai|hb|931@hbe|海北|HEB|haibei|hb|932@hbi|鹤壁|HAF|hebi|hb|933@hcb|会昌北|XEG|huichangbei|hcb|934@hch|华城|VCQ|huacheng|hc|935@hch|河唇|HCZ|hechun|hc|936@hch|汉川|HCN|hanchuan|hc|937@hch|海城|HCT|haicheng|hc|938@hch|合川|WKW|hechuan|hc|939@hct|黑冲滩|HCJ|heichongtan|hct|940@hcu|黄村|HCP|huangcun|hc|941@hcx|海城西|HXT|haichengxi|hcx|942@hde|化德|HGC|huade|hd|943@hdo|洪洞|HDV|hongtong|hd|944@hes|霍尔果斯|HFR|huoerguosi|hegs|945@hfe|横峰|HFG|hengfeng|hf|946@hfw|韩府湾|HXJ|hanfuwan|hfw|947@hgu|汉沽|HGP|hangu|hg|948@hgy|黄瓜园|HYM|huangguayuan|hgy|949@hgz|红光镇|IGW|hongguangzhen|hgz|950@hhe|浑河|HHT|hunhe|hh|951@hhg|红花沟|VHD|honghuagou|hhg|952@hht|黄花筒|HUD|huanghuatong|hht|953@hjd|贺家店|HJJ|hejiadian|hjd|954@hji|和静|HJR|hejing|hj|955@hji|红江|HFM|hongjiang|hj|956@hji|黑井|HIM|heijing|hj|957@hji|获嘉|HJF|huojia|hj|958@hji|河津|HJV|hejin|hj|959@hji|涵江|HJS|hanjiang|hj|960@hji|华家|HJT|huajia|hj|961@hjq|杭锦后旗|HDC|hangjinhouqi|hjhq|962@hjx|河间西|HXP|hejianxi|hjx|963@hjz|花家庄|HJM|huajiazhuang|hjz|964@hkn|河口南|HKJ|hekounan|hkn|965@hko|湖口|HKG|hukou|hk|966@hko|黄口|KOH|huangkou|hk|967@hla|呼兰|HUB|hulan|hl|968@hlb|葫芦岛北|HPD|huludaobei|hldb|969@hlh|浩良河|HHB|haolianghe|hlh|970@hlh|哈拉海|HIT|halahai|hlh|971@hli|鹤立|HOB|heli|hl|972@hli|桦林|HIB|hualin|hl|973@hli|黄陵|ULY|huangling|hl|974@hli|海林|HRB|hailin|hl|975@hli|虎林|VLB|hulin|hl|976@hli|寒岭|HAT|hanling|hl|977@hlo|和龙|HLL|helong|hl|978@hlo|海龙|HIL|hailong|hl|979@hls|哈拉苏|HAX|halasu|hls|980@hlt|呼鲁斯太|VTJ|hulusitai|hlst|981@hlz|火连寨|HLT|huolianzhai|hlz|982@hme|黄梅|VEH|huangmei|hm|983@hmy|韩麻营|HYP|hanmaying|hmy|984@hnh|黄泥河|HHL|huangnihe|hnh|985@hni|海宁|HNH|haining|hn|986@hno|惠农|HMJ|huinong|hn|987@hpi|和平|VAQ|heping|hp|988@hpz|花棚子|HZM|huapengzi|hpz|989@hqi|花桥|VQH|huaqiao|hq|990@hqi|宏庆|HEY|hongqing|hq|991@hre|怀仁|HRV|huairen|hr|992@hro|华容|HRN|huarong|hr|993@hsb|华山北|HDY|huashanbei|hsb|994@hsd|黄松甸|HDL|huangsongdian|hsd|995@hsg|和什托洛盖|VSR|heshituoluogai|hstlg|996@hsh|红山|VSB|hongshan|hs|997@hsh|汉寿|VSQ|hanshou|hs|998@hsh|衡山|HSQ|hengshan|hs|999@hsh|黑水|HOT|heishui|hs|1000@hsh|惠山|VCH|huishan|hs|1001@hsh|虎什哈|HHP|hushiha|hsh|1002@hsp|红寺堡|HSJ|hongsipu|hsp|1003@hst|虎石台|HUT|hushitai|hst|1004@hsw|海石湾|HSO|haishiwan|hsw|1005@hsx|衡山西|HEQ|hengshanxi|hsx|1006@hsx|红砂岘|VSJ|hongshaxian|hsx|1007@hta|黑台|HQB|heitai|ht|1008@hta|桓台|VTK|huantai|ht|1009@hti|和田|VTR|hetian|ht|1010@hto|会同|VTQ|huitong|ht|1011@htz|海坨子|HZT|haituozi|htz|1012@hwa|黑旺|HWK|heiwang|hw|1013@hwa|海湾|RWH|haiwan|hw|1014@hxi|红星|VXB|hongxing|hx|1015@hxi|徽县|HYY|huixian|hx|1016@hxl|红兴隆|VHB|hongxinglong|hxl|1017@hxt|换新天|VTB|huanxintian|hxt|1018@hxt|红岘台|HTJ|hongxiantai|hxt|1019@hya|红彦|VIX|hongyan|hy|1020@hya|海晏|HFO|haiyan|hy|1021@hya|合阳|HAY|heyang|hy|1022@hyd|衡阳东|HVQ|hengyangdong|hyd|1023@hyi|华蓥|HUW|huaying|hy|1024@hyi|汉阴|HQY|hanyin|hy|1025@hyt|黄羊滩|HGJ|huangyangtan|hyt|1026@hyu|汉源|WHW|hanyuan|hy|1027@hyu|河源|VIQ|heyuan|hy|1028@hyu|花园|HUN|huayuan|hy|1029@hyu|湟源|HNO|huangyuan|hy|1030@hyz|黄羊镇|HYJ|huangyangzhen|hyz|1031@hzh|湖州|VZH|huzhou|hz|1032@hzh|化州|HZZ|huazhou|hz|1033@hzh|黄州|VON|huangzhou|hz|1034@hzh|霍州|HZV|huozhou|hz|1035@hzx|惠州西|VXQ|huizhouxi|hzx|1036@jba|巨宝|JRT|jubao|jb|1037@jbi|靖边|JIY|jingbian|jb|1038@jbt|金宝屯|JBD|jinbaotun|jbt|1039@jcb|晋城北|JEF|jinchengbei|jcb|1040@jch|金昌|JCJ|jinchang|jc|1041@jch|鄄城|JCK|juancheng|jc|1042@jch|交城|JNV|jiaocheng|jc|1043@jch|建昌|JFD|jianchang|jc|1044@jde|峻德|JDB|junde|jd|1045@jdi|井店|JFP|jingdian|jd|1046@jdo|鸡东|JOB|jidong|jd|1047@jdu|江都|UDH|jiangdu|jd|1048@jgs|鸡冠山|JST|jiguanshan|jgs|1049@jgt|金沟屯|VGP|jingoutun|jgt|1050@jha|静海|JHP|jinghai|jh|1051@jhe|金河|JHX|jinhe|jh|1052@jhe|锦河|JHB|jinhe|jh|1053@jhe|精河|JHR|jinghe|jh|1054@jhn|精河南|JIR|jinghenan|jhn|1055@jhu|江华|JHZ|jianghua|jh|1056@jhu|建湖|AJH|jianhu|jh|1057@jjg|纪家沟|VJD|jijiagou|jjg|1058@jji|晋江|JJS|jinjiang|jj|1059@jji|锦界|JEY|jinjie|jj|1060@jji|姜家|JJB|jiangjia|jj|1061@jji|江津|JJW|jiangjin|jj|1062@jke|金坑|JKT|jinkeng|jk|1063@jli|芨岭|JLJ|jiling|jl|1064@jmc|金马村|JMM|jinmacun|jmc|1065@jmd|江门东|JWQ|jiangmendong|jmd|1066@jme|角美|JES|jiaomei|jm|1067@jna|莒南|JOK|junan|jn|1068@jna|井南|JNP|jingnan|jn|1069@jou|建瓯|JVS|jianou|jo|1070@jpe|经棚|JPC|jingpeng|jp|1071@jqi|江桥|JQX|jiangqiao|jq|1072@jsa|九三|SSX|jiusan|js|1073@jsb|金山北|EGH|jinshanbei|jsb|1074@jsh|嘉善|JSH|jiashan|js|1075@jsh|京山|JCN|jingshan|js|1076@jsh|建始|JRN|jianshi|js|1077@jsh|稷山|JVV|jishan|js|1078@jsh|吉舒|JSL|jishu|js|1079@jsh|建设|JET|jianshe|js|1080@jsh|甲山|JOP|jiashan|js|1081@jsj|建三江|JIB|jiansanjiang|jsj|1082@jsn|嘉善南|EAH|jiashannan|jsn|1083@jst|金山屯|JTB|jinshantun|jst|1084@jst|江所田|JOM|jiangsuotian|jst|1085@jta|景泰|JTJ|jingtai|jt|1086@jtn|九台南|JNL|jiutainan|jtn|1087@jwe|吉文|JWX|jiwen|jw|1088@jxi|进贤|JUG|jinxian|jx|1089@jxi|莒县|JKK|juxian|jx|1090@jxi|嘉祥|JUK|jiaxiang|jx|1091@jxi|介休|JXV|jiexiu|jx|1092@jxi|嘉兴|JXH|jiaxing|jx|1093@jxi|井陉|JJP|jingxing|jx|1094@jxn|嘉兴南|EPH|jiaxingnan|jxn|1095@jxz|夹心子|JXT|jiaxinzi|jxz|1096@jya|姜堰|UEH|jiangyan|jy|1097@jya|简阳|JYW|jianyang|jy|1098@jya|揭阳|JRQ|jieyang|jy|1099@jya|建阳|JYS|jianyang|jy|1100@jye|巨野|JYK|juye|jy|1101@jyo|江永|JYZ|jiangyong|jy|1102@jyu|缙云|JYH|jinyun|jy|1103@jyu|靖远|JYJ|jingyuan|jy|1104@jyu|江源|SZL|jiangyuan|jy|1105@jyu|济源|JYF|jiyuan|jy|1106@jyx|靖远西|JXJ|jingyuanxi|jyx|1107@jzb|胶州北|JZK|jiaozhoubei|jzb|1108@jzd|焦作东|WEF|jiaozuodong|jzd|1109@jzh|金寨|JZH|jinzhai|jz|1110@jzh|靖州|JEQ|jingzhou|jz|1111@jzh|荆州|JBN|jingzhou|jz|1112@jzh|胶州|JXK|jiaozhou|jz|1113@jzh|晋州|JXP|jinzhou|jz|1114@jzn|锦州南|JOD|jinzhounan|jzn|1115@jzu|焦作|JOF|jiaozuo|jz|1116@jzw|旧庄窝|JVP|jiuzhuangwo|jzw|1117@jzz|金杖子|JYD|jinzhangzi|jzz|1118@kan|开安|KAT|kaian|ka|1119@kch|库车|KCR|kuche|kc|1120@kch|康城|KCP|kangcheng|kc|1121@kde|库都尔|KDX|kuduer|kde|1122@kdi|宽甸|KDT|kuandian|kd|1123@kdo|克东|KOB|kedong|kd|1124@kdz|昆都仑召|KDC|kundulunzhao|kdlz|1125@kji|开江|KAW|kaijiang|kj|1126@kjj|康金井|KJB|kangjinjing|kjj|1127@klq|喀喇其|KQX|kalaqi|klq|1128@klu|开鲁|KLC|kailu|kl|1129@kly|克拉玛依|KHR|kelamayi|klmy|1130@kpn|开平南|PVQ|kaipingnan|kpn|1131@kqi|口前|KQL|kouqian|kq|1132@ksh|昆山|KSH|kunshan|ks|1133@ksh|奎山|KAB|kuishan|ks|1134@ksh|克山|KSB|keshan|ks|1135@kxl|康熙岭|KXZ|kangxiling|kxl|1136@kya|昆阳|KAM|kunyang|ky|1137@kyh|克一河|KHX|keyihe|kyh|1138@kyx|开原西|KXT|kaiyuanxi|kyx|1139@kzh|康庄|KZP|kangzhuang|kz|1140@lbi|来宾|UBZ|laibin|lb|1141@lbi|老边|LLT|laobian|lb|1142@lbx|灵宝西|LPF|lingbaoxi|lbx|1143@lch|龙川|LUQ|longchuan|lc|1144@lch|乐昌|LCQ|lechang|lc|1145@lch|黎城|UCP|licheng|lc|1146@lch|聊城|UCK|liaocheng|lc|1147@lcu|蓝村|LCK|lancun|lc|1148@lda|两当|LDY|liangdang|ld|1149@ldo|林东|LRC|lindong|ld|1150@ldu|乐都|LDO|ledu|ld|1151@ldx|梁底下|LDP|liangdixia|ldx|1152@ldz|六道河子|LVP|liudaohezi|ldhz|1153@lfa|鲁番|LVM|lufan|lf|1154@lfa|廊坊|LJP|langfang|lf|1155@lfa|落垡|LOP|luofa|lf|1156@lfb|廊坊北|LFP|langfangbei|lfb|1157@lfu|老府|UFD|laofu|lf|1158@lga|兰岗|LNB|langang|lg|1159@lgd|龙骨甸|LGM|longgudian|lgd|1160@lgo|芦沟|LOM|lugou|lg|1161@lgo|龙沟|LGJ|longgou|lg|1162@lgu|拉古|LGB|lagu|lg|1163@lha|临海|UFH|linhai|lh|1164@lha|林海|LXX|linhai|lh|1165@lha|拉哈|LHX|laha|lh|1166@lha|凌海|JID|linghai|lh|1167@lhe|柳河|LNL|liuhe|lh|1168@lhe|六合|KLH|liuhe|lh|1169@lhu|龙华|LHP|longhua|lh|1170@lhy|滦河沿|UNP|luanheyan|lhy|1171@lhz|六合镇|LEX|liuhezhen|lhz|1172@ljd|亮甲店|LRT|liangjiadian|ljd|1173@ljd|刘家店|UDT|liujiadian|ljd|1174@ljh|刘家河|LVT|liujiahe|ljh|1175@lji|连江|LKS|lianjiang|lj|1176@lji|庐江|UJH|lujiang|lj|1177@lji|李家|LJB|lijia|lj|1178@lji|罗江|LJW|luojiang|lj|1179@lji|廉江|LJZ|lianjiang|lj|1180@lji|两家|UJT|liangjia|lj|1181@lji|龙江|LJX|longjiang|lj|1182@lji|龙嘉|UJL|longjia|lj|1183@ljk|莲江口|LHB|lianjiangkou|ljk|1184@ljl|蔺家楼|ULK|linjialou|ljl|1185@ljp|李家坪|LIJ|lijiaping|ljp|1186@lka|兰考|LKF|lankao|lk|1187@lko|林口|LKB|linkou|lk|1188@lkp|路口铺|LKQ|lukoupu|lkp|1189@lla|老莱|LAX|laolai|ll|1190@lli|拉林|LAB|lalin|ll|1191@lli|陆良|LRM|luliang|ll|1192@lli|龙里|LLW|longli|ll|1193@lli|临澧|LWQ|linli|ll|1194@lli|兰棱|LLB|lanling|ll|1195@lli|零陵|UWZ|lingling|ll|1196@llo|卢龙|UAP|lulong|ll|1197@lmd|喇嘛甸|LMX|lamadian|lmd|1198@lmd|里木店|LMB|limudian|lmd|1199@lme|洛门|LMJ|luomen|lm|1200@lna|龙南|UNG|longnan|ln|1201@lpi|梁平|UQW|liangping|lp|1202@lpi|罗平|LPM|luoping|lp|1203@lpl|落坡岭|LPP|luopoling|lpl|1204@lps|六盘山|UPJ|liupanshan|lps|1205@lps|乐平市|LPG|lepingshi|lps|1206@lqi|临清|UQK|linqing|lq|1207@lqs|龙泉寺|UQJ|longquansi|lqs|1208@lsb|乐山北|UTW|leshanbei|ls|1209@lsc|乐善村|LUM|leshancun|lsc|1210@lsd|冷水江东|UDQ|lengshuijiangdong|lsjd|1211@lsg|连山关|LGT|lianshanguan|lsg|1212@lsg|流水沟|USP|liushuigou|lsg|1213@lsh|丽水|USH|lishui|ls|1214@lsh|陵水|LIQ|lingshui|ls|1215@lsh|罗山|LRN|luoshan|ls|1216@lsh|鲁山|LAF|lushan|ls|1217@lsh|梁山|LMK|liangshan|ls|1218@lsh|灵石|LSV|lingshi|ls|1219@lsh|露水河|LUL|lushuihe|lsh|1220@lsh|庐山|LSG|lushan|ls|1221@lsp|林盛堡|LBT|linshengpu|lsp|1222@lst|柳树屯|LSD|liushutun|lst|1223@lsz|龙山镇|LAS|longshanzhen|lsz|1224@lsz|梨树镇|LSB|lishuzhen|lsz|1225@lsz|李石寨|LET|lishizhai|lsz|1226@lta|黎塘|LTZ|litang|lt|1227@lta|轮台|LAR|luntai|lt|1228@lta|芦台|LTP|lutai|lt|1229@ltb|龙塘坝|LBM|longtangba|ltb|1230@ltu|濑湍|LVZ|laituan|lt|1231@ltx|骆驼巷|LTJ|luotuoxiang|ltx|1232@lwa|李旺|VLJ|liwang|lw|1233@lwd|莱芜东|LWK|laiwudong|lwd|1234@lws|狼尾山|LRJ|langweishan|lws|1235@lwu|灵武|LNJ|lingwu|lw|1236@lwx|莱芜西|UXK|laiwuxi|lwx|1237@lxi|朗乡|LXB|langxiang|lx|1238@lxi|陇县|LXY|longxian|lx|1239@lxi|临湘|LXQ|linxiang|lx|1240@lxi|芦溪|LUG|luxi|lx|1241@lxi|莱西|LXK|laixi|lx|1242@lxi|林西|LXC|linxi|lx|1243@lxi|滦县|UXP|luanxian|lx|1244@lya|莱阳|LYK|laiyang|ly|1245@lya|略阳|LYY|lueyang|ly|1246@lya|辽阳|LYT|liaoyang|ly|1247@lyd|凌源东|LDD|lingyuandong|lyd|1248@lyd|临沂东|UYK|linyidong|lyd|1249@lyg|连云港|UIH|lianyungang|lyg|1250@lyi|临颍|LNF|linying|ly|1251@lyi|老营|LXL|laoying|ly|1252@lyo|龙游|LMH|longyou|ly|1253@lyu|罗源|LVS|luoyuan|ly|1254@lyu|林源|LYX|linyuan|ly|1255@lyu|涟源|LAQ|lianyuan|ly|1256@lyu|涞源|LYP|laiyuan|ly|1257@lyx|耒阳西|LPQ|leiyangxi|lyx|1258@lze|临泽|LEJ|linze|lz|1259@lzg|龙爪沟|LZT|longzhuagou|lzg|1260@lzh|雷州|UAQ|leizhou|lz|1261@lzh|六枝|LIW|liuzhi|lz|1262@lzh|鹿寨|LIZ|luzhai|lz|1263@lzh|来舟|LZS|laizhou|lz|1264@lzh|龙镇|LZA|longzhen|lz|1265@lzh|拉鲊|LEM|lazha|lz|1266@lzq|兰州新区|LQJ|lanzhouxinqu|lzxq|1267@mas|马鞍山|MAH|maanshan|mas|1268@mba|毛坝|MBY|maoba|mb|1269@mbg|毛坝关|MGY|maobaguan|mbg|1270@mcb|麻城北|MBN|machengbei|mcb|1271@mch|渑池|MCF|mianchi|mc|1272@mch|明城|MCL|mingcheng|mc|1273@mch|庙城|MAP|miaocheng|mc|1274@mcn|渑池南|MNF|mianchinan|mcn|1275@mcp|茅草坪|KPM|maocaoping|mcp|1276@mdh|猛洞河|MUQ|mengdonghe|mdh|1277@mds|磨刀石|MOB|modaoshi|mds|1278@mdu|弥渡|MDF|midu|md|1279@mes|帽儿山|MRB|maoershan|mes|1280@mga|明港|MGN|minggang|mg|1281@mhk|梅河口|MHL|meihekou|mhk|1282@mhu|马皇|MHZ|mahuang|mh|1283@mjg|孟家岗|MGB|mengjiagang|mjg|1284@mla|美兰|MHQ|meilan|ml|1285@mld|汨罗东|MQQ|miluodong|mld|1286@mlh|马莲河|MHB|malianhe|mlh|1287@mli|茅岭|MLZ|maoling|ml|1288@mli|庙岭|MLL|miaoling|ml|1289@mli|茂林|MLD|maolin|ml|1290@mli|穆棱|MLB|muling|ml|1291@mli|马林|MID|malin|ml|1292@mlo|马龙|MGM|malong|ml|1293@mlt|木里图|MUD|mulitu|mlt|1294@mlu|汨罗|MLQ|miluo|ml|1295@mnh|玛纳斯湖|MNR|manasihu|mnsh|1296@mni|冕宁|UGW|mianning|mn|1297@mpa|沐滂|MPQ|mupang|mp|1298@mqh|马桥河|MQB|maqiaohe|mqh|1299@mqi|闽清|MQS|minqing|mq|1300@mqu|民权|MQF|minquan|mq|1301@msh|明水河|MUT|mingshuihe|msh|1302@msh|麻山|MAB|mashan|ms|1303@msh|眉山|MSW|meishan|ms|1304@msw|漫水湾|MKW|manshuiwan|msw|1305@msz|茂舍祖|MOM|maoshezu|msz|1306@msz|米沙子|MST|mishazi|msz|1307@mta|马踏|PWQ|mata|mt|1308@mxi|美溪|MEB|meixi|mx|1309@mxi|勉县|MVY|mianxian|mx|1310@mya|麻阳|MVQ|mayang|my|1311@myb|密云北|MUP|miyunbei|myb|1312@myi|米易|MMW|miyi|my|1313@myu|麦园|MYS|maiyuan|my|1314@myu|墨玉|MUR|moyu|my|1315@mzh|庙庄|MZJ|miaozhuang|mz|1316@mzh|米脂|MEY|mizhi|mz|1317@mzh|明珠|MFQ|mingzhu|mz|1318@nan|宁安|NAB|ningan|na|1319@nan|农安|NAT|nongan|na|1320@nbs|南博山|NBK|nanboshan|nbs|1321@nch|南仇|NCK|nanqiu|nc|1322@ncs|南城司|NSP|nanchengsi|ncs|1323@ncu|宁村|NCZ|ningcun|nc|1324@nde|宁德|NES|ningde|nd|1325@ngc|南观村|NGP|nanguancun|ngc|1326@ngd|南宫东|NFP|nangongdong|ngd|1327@ngl|南关岭|NLT|nanguanling|ngl|1328@ngu|宁国|NNH|ningguo|ng|1329@nha|宁海|NHH|ninghai|nh|1330@nhb|南华北|NHS|nanhuabei|nhb|1331@nhc|南河川|NHJ|nanhechuan|nhc|1332@nhz|泥河子|NHD|nihezi|nhz|1333@nji|宁家|NVT|ningjia|nj|1334@nji|南靖|NJS|nanjing|nj|1335@nji|牛家|NJB|niujia|nj|1336@nji|能家|NJD|nengjia|nj|1337@nko|南口|NKP|nankou|nk|1338@nkq|南口前|NKT|nankouqian|nkq|1339@nla|南朗|NNQ|nanlang|nl|1340@nli|乃林|NLD|nailin|nl|1341@nlk|尼勒克|NIR|nileke|nlk|1342@nlu|那罗|ULZ|naluo|nl|1343@nlx|宁陵县|NLF|ninglingxian|nlx|1344@nma|奈曼|NMD|naiman|nm|1345@nmi|宁明|NMZ|ningming|nm|1346@nmu|南木|NMX|nanmu|nm|1347@npn|南平南|NNS|nanpingnan|npn|1348@npu|那铺|NPZ|napu|np|1349@nqi|南桥|NQD|nanqiao|nq|1350@nqu|那曲|NQO|naqu|nq|1351@nqu|暖泉|NQJ|nuanquan|nq|1352@nta|南台|NTT|nantai|nt|1353@nto|南头|NOQ|nantou|nt|1354@nwu|宁武|NWV|ningwu|nw|1355@nwz|南湾子|NWP|nanwanzi|nwz|1356@nxb|南翔北|NEH|nanxiangbei|nxb|1357@nxi|宁乡|NXQ|ningxiang|nx|1358@nxi|内乡|NXF|neixiang|nx|1359@nxt|牛心台|NXT|niuxintai|nxt|1360@nyu|南峪|NUP|nanyu|ny|1361@nzg|娘子关|NIP|niangziguan|nzg|1362@nzh|南召|NAF|nanzhao|nz|1363@nzm|南杂木|NZT|nanzamu|nzm|1364@pan|蓬安|PAW|pengan|pa|1365@pan|平安|PAL|pingan|pa|1366@pay|平安驿|PNO|pinganyi|pay|1367@paz|磐安镇|PAJ|pananzhen|paz|1368@paz|平安镇|PZT|pinganzhen|paz|1369@pcd|蒲城东|PEY|puchengdong|pcd|1370@pch|蒲城|PCY|pucheng|pc|1371@pde|裴德|PDB|peide|pd|1372@pdi|偏店|PRP|piandian|pd|1373@pdx|平顶山西|BFF|pingdingshanxi|pdsx|1374@pdx|坡底下|PXJ|podixia|pdx|1375@pet|瓢儿屯|PRT|piaoertun|pet|1376@pfa|平房|PFB|pingfang|pf|1377@pga|平岗|PGL|pinggang|pg|1378@pgu|平关|PGM|pingguan|pg|1379@pgu|盘关|PAM|panguan|pg|1380@pgu|平果|PGZ|pingguo|pg|1381@phb|徘徊北|PHP|paihuaibei|phb|1382@phk|平河口|PHM|pinghekou|phk|1383@phu|平湖|PHQ|pinghu|ph|1384@pjb|盘锦北|PBD|panjinbei|pjb|1385@pjd|潘家店|PDP|panjiadian|pjd|1386@pkn|皮口南|PKT|pikounan|pk|1387@pld|普兰店|PLT|pulandian|pld|1388@pli|偏岭|PNT|pianling|pl|1389@psh|平山|PSB|pingshan|ps|1390@psh|彭山|PSW|pengshan|ps|1391@psh|皮山|PSR|pishan|ps|1392@psh|磐石|PSL|panshi|ps|1393@psh|平社|PSV|pingshe|ps|1394@psh|彭水|PHW|pengshui|ps|1395@pta|平台|PVT|pingtai|pt|1396@pti|平田|PTM|pingtian|pt|1397@pti|莆田|PTS|putian|pt|1398@ptq|葡萄菁|PTW|putaojing|ptq|1399@pwa|普湾|PWT|puwan|pw|1400@pwa|平旺|PWV|pingwang|pw|1401@pxg|平型关|PGV|pingxingguan|pxg|1402@pxi|普雄|POW|puxiong|px|1403@pxi|蓬溪|KZW|pengxi|px|1404@pxi|郫县|PWW|pixian|px|1405@pya|平洋|PYX|pingyang|py|1406@pya|彭阳|PYJ|pengyang|py|1407@pya|平遥|PYV|pingyao|py|1408@pyi|平邑|PIK|pingyi|py|1409@pyp|平原堡|PPJ|pingyuanpu|pyp|1410@pyu|平原|PYK|pingyuan|py|1411@pyu|平峪|PYP|pingyu|py|1412@pze|彭泽|PZG|pengze|pz|1413@pzh|邳州|PJH|pizhou|pz|1414@pzh|平庄|PZD|pingzhuang|pz|1415@pzi|泡子|POD|paozi|pz|1416@pzn|平庄南|PND|pingzhuangnan|pzn|1417@qan|乾安|QOT|qianan|qa|1418@qan|庆安|QAB|qingan|qa|1419@qan|迁安|QQP|qianan|qa|1420@qdb|祁东北|QRQ|qidongbei|qd|1421@qdi|七甸|QDM|qidian|qd|1422@qfd|曲阜东|QAK|qufudong|qfd|1423@qfe|庆丰|QFT|qingfeng|qf|1424@qft|奇峰塔|QVP|qifengta|qft|1425@qfu|曲阜|QFK|qufu|qf|1426@qha|琼海|QYQ|qionghai|qh|1427@qhd|秦皇岛|QTP|qinhuangdao|qhd|1428@qhe|千河|QUY|qianhe|qh|1429@qhe|清河|QIP|qinghe|qh|1430@qhm|清河门|QHD|qinghemen|qhm|1431@qhy|清华园|QHP|qinghuayuan|qhy|1432@qji|全椒|INH|quanjiao|qj|1433@qji|渠旧|QJZ|qujiu|qj|1434@qji|潜江|QJN|qianjiang|qj|1435@qji|秦家|QJB|qinjia|qj|1436@qji|綦江|QJW|qijiang|qj|1437@qjp|祁家堡|QBT|qijiapu|qjp|1438@qjx|清涧县|QNY|qingjianxian|qjx|1439@qjz|秦家庄|QZV|qinjiazhuang|qjz|1440@qlh|七里河|QLD|qilihe|qlh|1441@qli|秦岭|QLY|qinling|ql|1442@qli|渠黎|QLZ|quli|ql|1443@qlo|青龙|QIB|qinglong|ql|1444@qls|青龙山|QGH|qinglongshan|qls|1445@qme|祁门|QIH|qimen|qm|1446@qmt|前磨头|QMP|qianmotou|qmt|1447@qsh|青山|QSB|qingshan|qs|1448@qsh|确山|QSN|queshan|qs|1449@qsh|前山|QXQ|qianshan|qs|1450@qsh|清水|QUJ|qingshui|qs|1451@qsy|戚墅堰|QYH|qishuyan|qsy|1452@qti|青田|QVH|qingtian|qt|1453@qto|桥头|QAT|qiaotou|qt|1454@qtx|青铜峡|QTJ|qingtongxia|qtx|1455@qwe|前卫|QWD|qianwei|qw|1456@qwt|前苇塘|QWP|qianweitang|qwt|1457@qxi|渠县|QRW|quxian|qx|1458@qxi|祁县|QXV|qixian|qx|1459@qxi|青县|QXP|qingxian|qx|1460@qxi|桥西|QXJ|qiaoxi|qx|1461@qxu|清徐|QUV|qingxu|qx|1462@qxy|旗下营|QXC|qixiaying|qxy|1463@qya|千阳|QOY|qianyang|qy|1464@qya|沁阳|QYF|qinyang|qy|1465@qya|泉阳|QYL|quanyang|qy|1466@qyb|祁阳北|QVQ|qiyangbei|qy|1467@qyi|七营|QYJ|qiying|qy|1468@qys|庆阳山|QSJ|qingyangshan|qys|1469@qyu|清远|QBQ|qingyuan|qy|1470@qyu|清原|QYT|qingyuan|qy|1471@qzd|钦州东|QDZ|qinzhoudong|qzd|1472@qzh|钦州|QRZ|qinzhou|qz|1473@qzs|青州市|QZK|qingzhoushi|qzs|1474@ran|瑞安|RAH|ruian|ra|1475@rch|荣昌|RCW|rongchang|rc|1476@rch|瑞昌|RCG|ruichang|rc|1477@rga|如皋|RBH|rugao|rg|1478@rgu|容桂|RUQ|ronggui|rg|1479@rqi|任丘|RQP|renqiu|rq|1480@rsh|乳山|ROK|rushan|rs|1481@rsh|融水|RSZ|rongshui|rs|1482@rsh|热水|RSD|reshui|rs|1483@rxi|容县|RXZ|rongxian|rx|1484@rya|饶阳|RVP|raoyang|ry|1485@rya|汝阳|RYF|ruyang|ry|1486@ryh|绕阳河|RHD|raoyanghe|ryh|1487@rzh|汝州|ROF|ruzhou|rz|1488@sba|石坝|OBJ|shiba|sb|1489@sbc|上板城|SBP|shangbancheng|sbc|1490@sbi|施秉|AQW|shibing|sb|1491@sbn|上板城南|OBP|shangbanchengnan|sbcn|1492@sby|世博园|ZWT|shiboyuan|sby|1493@scb|双城北|SBB|shuangchengbei|scb|1494@sch|舒城|OCH|shucheng|sc|1495@sch|商城|SWN|shangcheng|sc|1496@sch|莎车|SCR|shache|sc|1497@sch|顺昌|SCS|shunchang|sc|1498@sch|神池|SMV|shenchi|sc|1499@sch|沙城|SCP|shacheng|sc|1500@sch|石城|SCT|shicheng|sc|1501@scz|山城镇|SCL|shanchengzhen|scz|1502@sda|山丹|SDJ|shandan|sd|1503@sde|顺德|ORQ|shunde|sd|1504@sde|绥德|ODY|suide|sd|1505@sdo|水洞|SIL|shuidong|sd|1506@sdu|商都|SXC|shangdu|sd|1507@sdu|十渡|SEP|shidu|sd|1508@sdw|四道湾|OUD|sidaowan|sdw|1509@sdy|顺德学院|OJQ|shundexueyuan|sdxy|1510@sfa|绅坊|OLH|shenfang|sf|1511@sfe|双丰|OFB|shuangfeng|sf|1512@sft|四方台|STB|sifangtai|sft|1513@sfu|水富|OTW|shuifu|sf|1514@sgk|三关口|OKJ|sanguankou|sgk|1515@sgl|桑根达来|OGC|sanggendalai|sgdl|1516@sgu|韶关|SNQ|shaoguan|sg|1517@sgz|上高镇|SVK|shanggaozhen|sgz|1518@sha|上杭|JBS|shanghang|sh|1519@sha|沙海|SED|shahai|sh|1520@she|蜀河|SHY|shuhe|sh|1521@she|松河|SBM|songhe|sh|1522@she|沙河|SHP|shahe|sh|1523@shk|沙河口|SKT|shahekou|shk|1524@shl|赛汗塔拉|SHC|saihantala|shtl|1525@shs|沙河市|VOP|shaheshi|shs|1526@shs|沙后所|SSD|shahousuo|shs|1527@sht|山河屯|SHL|shanhetun|sht|1528@shx|三河县|OXP|sanhexian|shx|1529@shy|四合永|OHD|siheyong|shy|1530@shz|三汇镇|OZW|sanhuizhen|shz|1531@shz|双河镇|SEL|shuanghezhen|shz|1532@shz|石河子|SZR|shihezi|shz|1533@shz|三合庄|SVP|sanhezhuang|shz|1534@sjd|三家店|ODP|sanjiadian|sjd|1535@sjh|水家湖|SQH|shuijiahu|sjh|1536@sjh|沈家河|OJJ|shenjiahe|sjh|1537@sjh|松江河|SJL|songjianghe|sjh|1538@sji|尚家|SJB|shangjia|sj|1539@sji|孙家|SUB|sunjia|sj|1540@sji|沈家|OJB|shenjia|sj|1541@sji|双吉|SML|shuangji|sj|1542@sji|松江|SAH|songjiang|sj|1543@sjk|三江口|SKD|sanjiangkou|sjk|1544@sjl|司家岭|OLK|sijialing|sjl|1545@sjn|松江南|IMH|songjiangnan|sjn|1546@sjn|石景山南|SRP|shijingshannan|sjsn|1547@sjt|邵家堂|SJJ|shaojiatang|sjt|1548@sjx|三江县|SOZ|sanjiangxian|sjx|1549@sjz|三家寨|SMM|sanjiazhai|sjz|1550@sjz|十家子|SJD|shijiazi|sjz|1551@sjz|松江镇|OZL|songjiangzhen|sjz|1552@sjz|施家嘴|SHM|shijiazui|sjz|1553@sjz|深井子|SWT|shenjingzi|sjz|1554@sld|什里店|OMP|shilidian|sld|1555@sle|疏勒|SUR|shule|sl|1556@slh|疏勒河|SHJ|shulehe|slh|1557@slh|舍力虎|VLD|shelihu|slh|1558@sli|石磷|SPB|shilin|sl|1559@sli|石林|SLM|shilin|sl|1560@sli|双辽|ZJD|shuangliao|sl|1561@sli|绥棱|SIB|suiling|sl|1562@sli|石岭|SOL|shiling|sl|1563@sln|石林南|LNM|shilinnan|sln|1564@slo|石龙|SLQ|shilong|sl|1565@slq|萨拉齐|SLC|salaqi|slq|1566@slu|索伦|SNT|suolun|sl|1567@slu|商洛|OLY|shangluo|sl|1568@slz|沙岭子|SLP|shalingzi|slz|1569@smb|石门县北|VFQ|shimenxianbei|smxb|1570@smn|三门峡南|SCF|sanmenxianan|smxn|1571@smx|三门县|OQH|sanmenxian|smx|1572@smx|石门县|OMQ|shimenxian|smx|1573@smx|三门峡西|SXF|sanmenxiaxi|smxx|1574@sni|肃宁|SYP|suning|sn|1575@son|宋|SOB|song|son|1576@spa|双牌|SBZ|shuangpai|sp|1577@spb|沙坪坝|CYW|shapingba|spb|1578@spd|四平东|PPT|sipingdong|spd|1579@spi|遂平|SON|suiping|sp|1580@spt|沙坡头|SFJ|shapotou|spt|1581@sqi|沙桥|SQM|shaqiao|sq|1582@sqn|商丘南|SPF|shangqiunan|sqn|1583@squ|水泉|SID|shuiquan|sq|1584@sqx|石泉县|SXY|shiquanxian|sqx|1585@sqz|石桥子|SQT|shiqiaozi|sqz|1586@src|石人城|SRB|shirencheng|src|1587@sre|石人|SRL|shiren|sr|1588@ssh|山市|SQB|shanshi|ss|1589@ssh|神树|SWB|shenshu|ss|1590@ssh|鄯善|SSR|shanshan|ss|1591@ssh|三水|SJQ|sanshui|ss|1592@ssh|泗水|OSK|sishui|ss|1593@ssh|石山|SAD|shishan|ss|1594@ssh|松树|SFT|songshu|ss|1595@ssh|首山|SAT|shoushan|ss|1596@ssj|三十家|SRD|sanshijia|ssj|1597@ssp|三十里堡|SST|sanshilipu|sslp|1598@ssz|双水镇|PQQ|shuangshuizhen|ssz|1599@ssz|松树镇|SSL|songshuzhen|ssz|1600@sta|松桃|MZQ|songtao|st|1601@sth|索图罕|SHX|suotuhan|sth|1602@stj|三堂集|SDH|santangji|stj|1603@sto|石头|OTB|shitou|st|1604@sto|神头|SEV|shentou|st|1605@stu|沙沱|SFM|shatuo|st|1606@swa|上万|SWP|shangwan|sw|1607@swu|孙吴|SKB|sunwu|sw|1608@swx|沙湾县|SXR|shawanxian|swx|1609@sxi|歙县|OVH|shexian|sx|1610@sxi|遂溪|SXZ|suixi|sx|1611@sxi|沙县|SAS|shaxian|sx|1612@sxi|绍兴|SOH|shaoxing|sx|1613@sxi|石岘|SXL|shixian|sx|1614@sxp|上西铺|SXM|shangxipu|sxp|1615@sxz|石峡子|SXJ|shixiazi|sxz|1616@sya|沭阳|FMH|shuyang|sy|1617@sya|绥阳|SYB|suiyang|sy|1618@sya|寿阳|SYV|shouyang|sy|1619@sya|水洋|OYP|shuiyang|sy|1620@syc|三阳川|SYJ|sanyangchuan|syc|1621@syd|上腰墩|SPJ|shangyaodun|syd|1622@syi|三营|OEJ|sanying|sy|1623@syi|顺义|SOP|shunyi|sy|1624@syj|三义井|OYD|sanyijing|syj|1625@syp|三源浦|SYL|sanyuanpu|syp|1626@syu|上虞|BDH|shangyu|sy|1627@syu|三原|SAY|sanyuan|sy|1628@syu|上园|SUD|shangyuan|sy|1629@syu|水源|OYJ|shuiyuan|sy|1630@syz|桑园子|SAJ|sangyuanzi|syz|1631@szb|绥中北|SND|suizhongbei|szb|1632@szb|苏州北|OHH|suzhoubei|szb|1633@szd|宿州东|SRH|suzhoudong|szd|1634@szd|深圳东|BJQ|shenzhendong|szd|1635@szh|深州|OZP|shenzhou|sz|1636@szh|孙镇|OZY|sunzhen|sz|1637@szh|绥中|SZD|suizhong|sz|1638@szh|尚志|SZB|shangzhi|sz|1639@szh|师庄|SNM|shizhuang|sz|1640@szi|松滋|SIN|songzi|sz|1641@szo|师宗|SEM|shizong|sz|1642@szq|苏州园区|KAH|suzhouyuanqu|szyq|1643@szq|苏州新区|ITH|suzhouxinqu|szxq|1644@tan|泰安|TMK|taian|ta|1645@tan|台安|TID|taian|ta|1646@tay|通安驿|TAJ|tonganyi|tay|1647@tba|桐柏|TBF|tongbai|tb|1648@tbe|通北|TBB|tongbei|tb|1649@tch|桐城|TTH|tongcheng|tc|1650@tch|汤池|TCX|tangchi|tc|1651@tch|郯城|TZK|tancheng|tc|1652@tch|铁厂|TCL|tiechang|tc|1653@tcu|桃村|TCK|taocun|tc|1654@tda|通道|TRQ|tongdao|td|1655@tdo|田东|TDZ|tiandong|td|1656@tga|天岗|TGL|tiangang|tg|1657@tgl|土贵乌拉|TGC|tuguiwula|tgwl|1658@tgo|通沟|TOL|tonggou|tg|1659@tgu|太谷|TGV|taigu|tg|1660@tha|塔哈|THX|taha|th|1661@tha|棠海|THM|tanghai|th|1662@the|唐河|THF|tanghe|th|1663@the|泰和|THG|taihe|th|1664@thu|太湖|TKH|taihu|th|1665@tji|团结|TIX|tuanjie|tj|1666@tjj|谭家井|TNJ|tanjiajing|tjj|1667@tjt|陶家屯|TOT|taojiatun|tjt|1668@tjw|唐家湾|PDQ|tangjiawan|tjw|1669@tjz|统军庄|TZP|tongjunzhuang|tjz|1670@tld|吐列毛杜|TMD|tuliemaodu|tlmd|1671@tlh|图里河|TEX|tulihe|tlh|1672@tli|铜陵|TJH|tongling|tl|1673@tli|田林|TFZ|tianlin|tl|1674@tli|亭亮|TIZ|tingliang|tl|1675@tli|铁力|TLB|tieli|tl|1676@tlx|铁岭西|PXT|tielingxi|tlx|1677@tmb|图们北|QSL|tumenbei|tmb|1678@tme|天门|TMN|tianmen|tm|1679@tmn|天门南|TNN|tianmennan|tmn|1680@tms|太姥山|TLS|taimushan|tms|1681@tmt|土牧尔台|TRC|tumuertai|tmet|1682@tmz|土门子|TCJ|tumenzi|tmz|1683@tna|洮南|TVT|taonan|tn|1684@tna|潼南|TVW|tongnan|tn|1685@tpc|太平川|TIT|taipingchuan|tpc|1686@tpz|太平镇|TEB|taipingzhen|tpz|1687@tqi|图强|TQX|tuqiang|tq|1688@tqi|台前|TTK|taiqian|tq|1689@tql|天桥岭|TQL|tianqiaoling|tql|1690@tqz|土桥子|TQJ|tuqiaozi|tqz|1691@tsc|汤山城|TCT|tangshancheng|tsc|1692@tsh|桃山|TAB|taoshan|ts|1693@tsh|台山|PUQ|taishan|ts|1694@tsz|塔石嘴|TIM|tashizui|tsz|1695@ttu|通途|TUT|tongtu|tt|1696@twh|汤旺河|THB|tangwanghe|twh|1697@txi|同心|TXJ|tongxin|tx|1698@txi|土溪|TSW|tuxi|tx|1699@txi|桐乡|TCH|tongxiang|tx|1700@tya|田阳|TRZ|tianyang|ty|1701@tyi|天义|TND|tianyi|ty|1702@tyi|汤阴|TYF|tangyin|ty|1703@tyl|驼腰岭|TIL|tuoyaoling|tyl|1704@tys|太阳山|TYJ|taiyangshan|tys|1705@tyu|通榆|KTT|tongyu|ty|1706@tyu|汤原|TYB|tangyuan|ty|1707@tyy|塔崖驿|TYP|tayayi|tyy|1708@tzd|滕州东|TEK|tengzhoudong|tzd|1709@tzh|台州|TZH|taizhou|tz|1710@tzh|天祝|TZJ|tianzhu|tz|1711@tzh|滕州|TXK|tengzhou|tz|1712@tzh|天镇|TZV|tianzhen|tz|1713@tzl|桐子林|TEW|tongzilin|tzl|1714@tzs|天柱山|QWH|tianzhushan|tzs|1715@wan|文安|WBP|wenan|wa|1716@wan|武安|WAP|wuan|wa|1717@waz|王安镇|WVP|wanganzhen|waz|1718@wbu|吴堡|WUY|wubu|wb|1719@wca|旺苍|WEW|wangcang|wc|1720@wcg|五叉沟|WCT|wuchagou|wcg|1721@wch|文昌|WEQ|wenchang|wc|1722@wch|温春|WDB|wenchun|wc|1723@wdc|五大连池|WRB|wudalianchi|wdlc|1724@wde|文登|WBK|wendeng|wd|1725@wdg|五道沟|WDL|wudaogou|wdg|1726@wdh|五道河|WHP|wudaohe|wdh|1727@wdi|文地|WNZ|wendi|wd|1728@wdo|卫东|WVT|weidong|wd|1729@wds|武当山|WRN|wudangshan|wds|1730@wdu|望都|WDP|wangdu|wd|1731@weh|乌尔旗汗|WHX|wuerqihan|weqh|1732@wfa|潍坊|WFK|weifang|wf|1733@wft|万发屯|WFB|wanfatun|wft|1734@wfu|王府|WUT|wangfu|wf|1735@wfx|瓦房店西|WXT|wafangdianxi|wfdx|1736@wga|王岗|WGB|wanggang|wg|1737@wgo|武功|WGY|wugong|wg|1738@wgo|湾沟|WGL|wangou|wg|1739@wgt|吴官田|WGM|wuguantian|wgt|1740@wha|乌海|WVC|wuhai|wh|1741@whe|苇河|WHB|weihe|wh|1742@whu|卫辉|WHF|weihui|wh|1743@wjc|吴家川|WCJ|wujiachuan|wjc|1744@wji|五家|WUB|wujia|wj|1745@wji|威箐|WAM|weiqing|wj|1746@wji|午汲|WJP|wuji|wj|1747@wji|渭津|WJL|weijin|wj|1748@wjw|王家湾|WJJ|wangjiawan|wjw|1749@wke|倭肯|WQB|woken|wk|1750@wks|五棵树|WKT|wukeshu|wks|1751@wlb|五龙背|WBT|wulongbei|wlb|1752@wld|乌兰哈达|WLC|wulanhada|wlhd|1753@wle|万乐|WEB|wanle|wl|1754@wlg|瓦拉干|WVX|walagan|wlg|1755@wli|温岭|VHH|wenling|wl|1756@wli|五莲|WLK|wulian|wl|1757@wlq|乌拉特前旗|WQC|wulateqianqi|wltqq|1758@wls|乌拉山|WSC|wulashan|wls|1759@wlt|卧里屯|WLX|wolitun|wlt|1760@wnb|渭南北|WBY|weinanbei|wnb|1761@wne|乌奴耳|WRX|wunuer|wne|1762@wni|万宁|WNQ|wanning|wn|1763@wni|万年|WWG|wannian|wn|1764@wnn|渭南南|WVY|weinannan|wnn|1765@wnz|渭南镇|WNJ|weinanzhen|wnz|1766@wpi|沃皮|WPT|wopi|wp|1767@wqi|吴桥|WUP|wuqiao|wq|1768@wqi|汪清|WQL|wangqing|wq|1769@wqi|武清|WWP|wuqing|wq|1770@wsh|武山|WSJ|wushan|ws|1771@wsh|文水|WEV|wenshui|ws|1772@wsz|魏善庄|WSP|weishanzhuang|wsz|1773@wto|王瞳|WTP|wangtong|wt|1774@wts|五台山|WSV|wutaishan|wts|1775@wtz|王团庄|WZJ|wangtuanzhuang|wtz|1776@wwu|五五|WVR|wuwu|ww|1777@wxd|无锡东|WGH|wuxidong|wxd|1778@wxi|卫星|WVB|weixing|wx|1779@wxi|闻喜|WXV|wenxi|wx|1780@wxi|武乡|WVV|wuxiang|wx|1781@wxq|无锡新区|IFH|wuxixinqu|wxxq|1782@wxu|武穴|WXN|wuxue|wx|1783@wxu|吴圩|WYZ|wuxu|wx|1784@wya|王杨|WYB|wangyang|wy|1785@wyi|武义|RYH|wuyi|wy|1786@wyi|五营|WWB|wuying|wy|1787@wyt|瓦窑田|WIM|wayaotian|wyt|1788@wyu|五原|WYC|wuyuan|wy|1789@wzg|苇子沟|WZL|weizigou|wzg|1790@wzh|韦庄|WZY|weizhuang|wz|1791@wzh|五寨|WZV|wuzhai|wz|1792@wzt|王兆屯|WZB|wangzhaotun|wzt|1793@wzz|微子镇|WQP|weizizhen|wzz|1794@wzz|魏杖子|WKD|weizhangzi|wzz|1795@xan|新安|EAM|xinan|xa|1796@xan|兴安|XAZ|xingan|xa|1797@xax|新安县|XAF|xinanxian|xax|1798@xba|新保安|XAP|xinbaoan|xba|1799@xbc|下板城|EBP|xiabancheng|xbc|1800@xbl|西八里|XLP|xibali|xbl|1801@xch|宣城|ECH|xuancheng|xc|1802@xch|兴城|XCD|xingcheng|xc|1803@xcu|小村|XEM|xiaocun|xc|1804@xcy|新绰源|XRX|xinchuoyuan|xcy|1805@xcz|下城子|XCB|xiachengzi|xcz|1806@xcz|新城子|XCT|xinchengzi|xcz|1807@xde|喜德|EDW|xide|xd|1808@xdj|小得江|EJM|xiaodejiang|xdj|1809@xdm|西大庙|XMP|xidamiao|xdm|1810@xdo|小董|XEZ|xiaodong|xd|1811@xdo|小东|XOD|xiaodong|xd|1812@xfa|香坊|XFB|xiangfang|xf|1813@xfe|信丰|EFG|xinfeng|xf|1814@xfe|襄汾|XFV|xiangfen|xf|1815@xfe|息烽|XFW|xifeng|xf|1816@xga|新干|EGG|xingan|xg|1817@xga|轩岗|XGV|xuangang|xg|1818@xga|孝感|XGN|xiaogan|xg|1819@xgc|西固城|XUJ|xigucheng|xgc|1820@xgu|兴国|EUG|xingguo|xg|1821@xgu|西固|XIJ|xigu|xg|1822@xgy|夏官营|XGJ|xiaguanying|xgy|1823@xgz|西岗子|NBB|xigangzi|xgz|1824@xha|宣汉|XHY|xuanhan|xh|1825@xhe|襄河|XXB|xianghe|xh|1826@xhe|新和|XIR|xinhe|xh|1827@xhe|宣和|XWJ|xuanhe|xh|1828@xhj|斜河涧|EEP|xiehejian|xhj|1829@xht|新华屯|XAX|xinhuatun|xht|1830@xhu|新会|EFQ|xinhui|xh|1831@xhu|新华|XHB|xinhua|xh|1832@xhu|新晃|XLQ|xinhuang|xh|1833@xhu|新化|EHQ|xinhua|xh|1834@xhu|宣化|XHP|xuanhua|xh|1835@xhx|兴和西|XEC|xinghexi|xhx|1836@xhy|小河沿|XYD|xiaoheyan|xhy|1837@xhy|下花园|XYP|xiahuayuan|xhy|1838@xhz|小河镇|EKY|xiaohezhen|xhz|1839@xjd|徐家店|HYK|xujiadian|xjd|1840@xji|徐家|XJB|xujia|xj|1841@xji|峡江|EJG|xiajiang|xj|1842@xji|新绛|XJV|xinjiang|xj|1843@xji|辛集|ENP|xinji|xj|1844@xji|新江|XJM|xinjiang|xj|1845@xjk|西街口|EKM|xijiekou|xjk|1846@xjt|许家屯|XJT|xujiatun|xjt|1847@xjt|许家台|XTJ|xujiatai|xjt|1848@xjz|谢家镇|XMT|xiejiazhen|xjz|1849@xka|兴凯|EKB|xingkai|xk|1850@xla|小榄|EAQ|xiaolan|xl|1851@xla|香兰|XNB|xianglan|xl|1852@xld|兴隆店|XDD|xinglongdian|xld|1853@xle|新乐|ELP|xinle|xl|1854@xli|新林|XPX|xinlin|xl|1855@xli|小岭|XLB|xiaoling|xl|1856@xli|新李|XLJ|xinli|xl|1857@xli|西林|XYB|xilin|xl|1858@xli|西柳|GCT|xiliu|xl|1859@xli|仙林|XPH|xianlin|xl|1860@xlt|新立屯|XLD|xinlitun|xlt|1861@xlx|兴隆县|EXP|xinglongxian|xlx|1862@xlz|兴隆镇|XZB|xinglongzhen|xlz|1863@xlz|新立镇|XGT|xinlizhen|xlz|1864@xmi|新民|XMD|xinmin|xm|1865@xms|西麻山|XMB|ximashan|xms|1866@xmt|下马塘|XAT|xiamatang|xmt|1867@xna|孝南|XNV|xiaonan|xn|1868@xnb|咸宁北|XRN|xianningbei|xnb|1869@xni|兴宁|ENQ|xingning|xn|1870@xni|咸宁|XNN|xianning|xn|1871@xpd|犀浦东|XAW|xipudong|xpd|1872@xpi|西平|XPN|xiping|xp|1873@xpi|兴平|XPY|xingping|xp|1874@xpt|新坪田|XPM|xinpingtian|xpt|1875@xpu|霞浦|XOS|xiapu|xp|1876@xpu|溆浦|EPQ|xupu|xp|1877@xpu|犀浦|XIW|xipu|xp|1878@xqi|新青|XQB|xinqing|xq|1879@xqi|新邱|XQD|xinqiu|xq|1880@xqp|兴泉堡|XQJ|xingquanbu|xqp|1881@xrq|仙人桥|XRL|xianrenqiao|xrq|1882@xsg|小寺沟|ESP|xiaosigou|xsg|1883@xsh|杏树|XSB|xingshu|xs|1884@xsh|浠水|XZN|xishui|xs|1885@xsh|下社|XSV|xiashe|xs|1886@xsh|小市|XST|xiaoshi|xs|1887@xsh|徐水|XSP|xushui|xs|1888@xsh|夏石|XIZ|xiashi|xs|1889@xsh|小哨|XAM|xiaoshao|xs|1890@xsh|秀山|ETW|xiushan|xs|1891@xsp|新松浦|XOB|xinsongpu|xsp|1892@xst|杏树屯|XDT|xingshutun|xst|1893@xsw|许三湾|XSJ|xusanwan|xsw|1894@xta|湘潭|XTQ|xiangtan|xt|1895@xta|邢台|XTP|xingtai|xt|1896@xta|向塘|XTG|xiangtang|xt|1897@xtx|仙桃西|XAN|xiantaoxi|xtx|1898@xtz|下台子|EIP|xiataizi|xtz|1899@xwe|徐闻|XJQ|xuwen|xw|1900@xwp|新窝铺|EPD|xinwopu|xwp|1901@xwu|修武|XWF|xiuwu|xw|1902@xxi|新县|XSN|xinxian|xx|1903@xxi|息县|ENN|xixian|xx|1904@xxi|西乡|XQY|xixiang|xx|1905@xxi|湘乡|XXQ|xiangxiang|xx|1906@xxi|西峡|XIF|xixia|xx|1907@xxi|孝西|XOV|xiaoxi|xx|1908@xxj|小新街|XXM|xiaoxinjie|xxj|1909@xxx|新兴县|XGQ|xinxingxian|xxx|1910@xxz|西小召|XZC|xixiaozhao|xxz|1911@xxz|小西庄|XXP|xiaoxizhuang|xxz|1912@xya|向阳|XDB|xiangyang|xy|1913@xya|旬阳|XUY|xunyang|xy|1914@xyb|旬阳北|XBY|xunyangbei|xyb|1915@xyd|襄阳东|XWN|xiangyangdong|xyd|1916@xye|兴业|SNZ|xingye|xy|1917@xyg|小雨谷|XHM|xiaoyugu|xyg|1918@xyi|新沂|VIH|xinyi|xy|1919@xyi|兴义|XRZ|xingyi|xy|1920@xyi|信宜|EEQ|xinyi|xy|1921@xyj|小月旧|XFM|xiaoyuejiu|xyj|1922@xyq|小扬气|XYX|xiaoyangqi|xyq|1923@xyu|襄垣|EIF|xiangyuan|xy|1924@xyx|夏邑县|EJH|xiayixian|xyx|1925@xyx|祥云西|EXM|xiangyunxi|xyx|1926@xyy|新友谊|EYB|xinyouyi|xyy|1927@xyz|新阳镇|XZJ|xinyangzhen|xyz|1928@xzd|徐州东|UUH|xuzhoudong|xzd|1929@xzf|新帐房|XZX|xinzhangfang|xzf|1930@xzh|悬钟|XRP|xuanzhong|xz|1931@xzh|新肇|XZT|xinzhao|xz|1932@xzh|忻州|XXV|xinzhou|xz|1933@xzi|汐子|XZD|xizi|xz|1934@xzm|西哲里木|XRD|xizhelimu|xzlm|1935@xzz|新杖子|ERP|xinzhangzi|xzz|1936@yan|姚安|YAC|yaoan|ya|1937@yan|依安|YAX|yian|ya|1938@yan|永安|YAS|yongan|ya|1939@yax|永安乡|YNB|yonganxiang|yax|1940@ybl|亚布力|YBB|yabuli|ybl|1941@ybs|元宝山|YUD|yuanbaoshan|ybs|1942@yca|羊草|YAB|yangcao|yc|1943@ycd|秧草地|YKM|yangcaodi|ycd|1944@ych|阳澄湖|AIH|yangchenghu|ych|1945@ych|迎春|YYB|yingchun|yc|1946@ych|叶城|YER|yecheng|yc|1947@ych|盐池|YKJ|yanchi|yc|1948@ych|砚川|YYY|yanchuan|yc|1949@ych|阳春|YQQ|yangchun|yc|1950@ych|宜城|YIN|yicheng|yc|1951@ych|应城|YHN|yingcheng|yc|1952@ych|禹城|YCK|yucheng|yc|1953@ych|晏城|YEK|yancheng|yc|1954@ych|阳城|YNF|yangcheng|yc|1955@ych|阳岔|YAL|yangcha|yc|1956@ych|郓城|YPK|yuncheng|yc|1957@ych|雁翅|YAP|yanchi|yc|1958@ycl|云彩岭|ACP|yuncailing|ycl|1959@ycx|虞城县|IXH|yuchengxian|ycx|1960@ycz|营城子|YCT|yingchengzi|ycz|1961@yde|英德|YDQ|yingde|yd|1962@yde|永登|YDJ|yongdeng|yd|1963@ydi|尹地|YDM|yindi|yd|1964@ydi|永定|YGS|yongding|yd|1965@ydo|阳东|WLQ|yangdong|yd|1966@yds|雁荡山|YGH|yandangshan|yds|1967@ydu|于都|YDG|yudu|yd|1968@ydu|园墩|YAJ|yuandun|yd|1969@ydx|英德西|IIQ|yingdexi|ydx|1970@yfy|永丰营|YYM|yongfengying|yfy|1971@yga|杨岗|YRB|yanggang|yg|1972@yga|阳高|YOV|yanggao|yg|1973@ygu|阳谷|YIK|yanggu|yg|1974@yha|友好|YOB|youhao|yh|1975@yha|余杭|EVH|yuhang|yh|1976@yhc|沿河城|YHP|yanhecheng|yhc|1977@yhu|岩会|AEP|yanhui|yh|1978@yjh|羊臼河|YHM|yangjiuhe|yjh|1979@yji|永嘉|URH|yongjia|yj|1980@yji|营街|YAM|yingjie|yj|1981@yji|盐津|AEW|yanjin|yj|1982@yji|阳江|WRQ|yangjiang|yj|1983@yji|余江|YHG|yujiang|yj|1984@yji|燕郊|AJP|yanjiao|yj|1985@yji|姚家|YAT|yaojia|yj|1986@yjj|岳家井|YGJ|yuejiajing|yjj|1987@yjp|一间堡|YJT|yijianpu|yjp|1988@yjs|英吉沙|YIR|yingjisha|yjs|1989@yjs|云居寺|AFP|yunjusi|yjs|1990@yjz|燕家庄|AZK|yanjiazhuang|yjz|1991@yka|永康|RFH|yongkang|yk|1992@ykd|营口东|YGT|yingkoudong|ykd|1993@yla|银浪|YJX|yinlang|yl|1994@yla|永郎|YLW|yonglang|yl|1995@ylb|宜良北|YSM|yiliangbei|ylb|1996@yld|永乐店|YDY|yongledian|yld|1997@ylh|伊拉哈|YLX|yilaha|ylh|1998@yli|伊林|YLB|yilin|yl|1999@yli|杨陵|YSY|yangling|yl|2000@yli|彝良|ALW|yiliang|yl|2001@yli|杨林|YLM|yanglin|yl|2002@ylp|余粮堡|YLD|yuliangpu|ylp|2003@ylq|杨柳青|YQP|yangliuqing|ylq|2004@ylt|月亮田|YUM|yueliangtian|ylt|2005@yma|义马|YMF|yima|ym|2006@ymb|阳明堡|YVV|yangmingbu|ymb|2007@yme|玉门|YXJ|yumen|ym|2008@yme|云梦|YMN|yunmeng|ym|2009@ymo|元谋|YMM|yuanmou|ym|2010@yms|一面山|YST|yimianshan|yms|2011@yna|沂南|YNK|yinan|yn|2012@yna|宜耐|YVM|yinai|yn|2013@ynd|伊宁东|YNR|yiningdong|ynd|2014@yps|营盘水|YZJ|yingpanshui|yps|2015@ypu|羊堡|ABM|yangpu|yp|2016@yqb|阳泉北|YPP|yangquanbei|yqb|2017@yqi|乐清|UPH|yueqing|yq|2018@yqi|焉耆|YSR|yanqi|yq|2019@yqi|源迁|AQK|yuanqian|yq|2020@yqt|姚千户屯|YQT|yaoqianhutun|yqht|2021@yqu|阳曲|YQV|yangqu|yq|2022@ysg|榆树沟|YGP|yushugou|ysg|2023@ysh|月山|YBF|yueshan|ys|2024@ysh|玉石|YSJ|yushi|ys|2025@ysh|玉舍|AUM|yushe|ys|2026@ysh|偃师|YSF|yanshi|ys|2027@ysh|沂水|YUK|yishui|ys|2028@ysh|榆社|YSV|yushe|ys|2029@ysh|颍上|YVH|yingshang|ys|2030@ysh|窑上|ASP|yaoshang|ys|2031@ysh|元氏|YSP|yuanshi|ys|2032@ysl|杨树岭|YAD|yangshuling|ysl|2033@ysp|野三坡|AIP|yesanpo|ysp|2034@yst|榆树屯|YSX|yushutun|yst|2035@yst|榆树台|YUT|yushutai|yst|2036@ysz|鹰手营子|YIP|yingshouyingzi|ysyz|2037@yta|源潭|YTQ|yuantan|yt|2038@ytp|牙屯堡|YTZ|yatunpu|ytp|2039@yts|烟筒山|YSL|yantongshan|yts|2040@ytt|烟筒屯|YUX|yantongtun|ytt|2041@yws|羊尾哨|YWM|yangweishao|yws|2042@yxi|越西|YHW|yuexi|yx|2043@yxi|攸县|YOG|youxian|yx|2044@yxi|阳西|WMQ|yangxi|yx|2045@yxi|永修|ACG|yongxiu|yx|2046@yxx|玉溪西|YXM|yuxixi|yxx|2047@yya|弋阳|YIG|yiyang|yy|2048@yya|余姚|YYH|yuyao|yy|2049@yya|酉阳|AFW|youyang|yy|2050@yyd|岳阳东|YIQ|yueyangdong|yyd|2051@yyi|阳邑|ARP|yangyi|yy|2052@yyu|鸭园|YYL|yayuan|yy|2053@yyz|鸳鸯镇|YYJ|yuanyangzhen|yyz|2054@yzb|燕子砭|YZY|yanzibian|yzb|2055@yzh|仪征|UZH|yizheng|yz|2056@yzh|宜州|YSZ|yizhou|yz|2057@yzh|兖州|YZK|yanzhou|yz|2058@yzi|迤资|YQM|yizi|yz|2059@yzw|羊者窝|AEM|yangzhewo|yzw|2060@yzz|杨杖子|YZD|yangzhangzi|yzz|2061@zan|镇安|ZEY|zhenan|za|2062@zan|治安|ZAD|zhian|za|2063@zba|招柏|ZBP|zhaobai|zb|2064@zbw|张百湾|ZUP|zhangbaiwan|zbw|2065@zcc|中川机场|ZJJ|zhongchuanjichang|zcjc|2066@zch|枝城|ZCN|zhicheng|zc|2067@zch|子长|ZHY|zichang|zc|2068@zch|诸城|ZQK|zhucheng|zc|2069@zch|邹城|ZIK|zoucheng|zc|2070@zch|赵城|ZCV|zhaocheng|zc|2071@zda|章党|ZHT|zhangdang|zd|2072@zdi|正定|ZDP|zhengding|zd|2073@zdo|肇东|ZDB|zhaodong|zd|2074@zfp|照福铺|ZFM|zhaofupu|zfp|2075@zgt|章古台|ZGD|zhanggutai|zgt|2076@zgu|赵光|ZGB|zhaoguang|zg|2077@zhe|中和|ZHX|zhonghe|zh|2078@zhm|中华门|VNH|zhonghuamen|zhm|2079@zjb|枝江北|ZIN|zhijiangbei|zjb|2080@zjc|钟家村|ZJY|zhongjiacun|zjc|2081@zjg|朱家沟|ZUB|zhujiagou|zjg|2082@zjg|紫荆关|ZYP|zijingguan|zjg|2083@zji|周家|ZOB|zhoujia|zj|2084@zji|诸暨|ZDH|zhuji|zj|2085@zjn|镇江南|ZEH|zhenjiangnan|zjn|2086@zjt|周家屯|ZOD|zhoujiatun|zjt|2087@zjw|褚家湾|CWJ|zhujiawan|zjw|2088@zjx|湛江西|ZWQ|zhanjiangxi|zjx|2089@zjy|朱家窑|ZUJ|zhujiayao|zjy|2090@zjz|曾家坪子|ZBW|zengjiapingzi|zjpz|2091@zla|张兰|ZLV|zhanglan|zl|2092@zla|镇赉|ZLT|zhenlai|zl|2093@zli|枣林|ZIV|zaolin|zl|2094@zlt|扎鲁特|ZLD|zhalute|zlt|2095@zlx|扎赉诺尔西|ZXX|zhalainuoerxi|zlrex|2096@zmt|樟木头|ZOQ|zhangmutou|zmt|2097@zmu|中牟|ZGF|zhongmu|zm|2098@znd|中宁东|ZDJ|zhongningdong|znd|2099@zni|中宁|VNJ|zhongning|zn|2100@znn|中宁南|ZNJ|zhongningnan|znn|2101@zpi|镇平|ZPF|zhenping|zp|2102@zpi|漳平|ZPS|zhangping|zp|2103@zpu|泽普|ZPR|zepu|zp|2104@zqi|枣强|ZVP|zaoqiang|zq|2105@zqi|张桥|ZQY|zhangqiao|zq|2106@zqi|章丘|ZTK|zhangqiu|zq|2107@zrh|朱日和|ZRC|zhurihe|zrh|2108@zrl|泽润里|ZLM|zerunli|zrl|2109@zsb|中山北|ZGQ|zhongshanbei|zsb|2110@zsd|樟树东|ZOG|zhangshudong|zsd|2111@zsh|珠斯花|ZHD|zhusihua|zsh|2112@zsh|中山|ZSQ|zhongshan|zs|2113@zsh|柞水|ZSY|zhashui|zs|2114@zsh|钟山|ZSZ|zhongshan|zs|2115@zsh|樟树|ZSG|zhangshu|zs|2116@zwo|珠窝|ZOP|zhuwo|zw|2117@zwt|张维屯|ZWB|zhangweitun|zwt|2118@zwu|彰武|ZWD|zhangwu|zw|2119@zxi|棕溪|ZOY|zongxi|zx|2120@zxi|钟祥|ZTN|zhongxiang|zx|2121@zxi|资溪|ZXS|zixi|zx|2122@zxi|镇西|ZVT|zhenxi|zx|2123@zxi|张辛|ZIP|zhangxin|zx|2124@zxq|正镶白旗|ZXC|zhengxiangbaiqi|zxbq|2125@zya|紫阳|ZVY|ziyang|zy|2126@zya|枣阳|ZYN|zaoyang|zy|2127@zyb|竹园坝|ZAW|zhuyuanba|zyb|2128@zye|张掖|ZYJ|zhangye|zy|2129@zyu|镇远|ZUW|zhenyuan|zy|2130@zzd|漳州东|GOS|zhangzhoudong|zzd|2131@zzh|漳州|ZUS|zhangzhou|zz|2132@zzh|壮志|ZUX|zhuangzhi|zz|2133@zzh|子洲|ZZY|zizhou|zz|2134@zzh|中寨|ZZM|zhongzhai|zz|2135@zzh|涿州|ZXP|zhuozhou|zz|2136@zzi|咋子|ZAL|zhazi|zz|2137@zzs|卓资山|ZZC|zhuozishan|zzs|2138@zzx|株洲西|ZAQ|zhuzhouxi|zzx|2139@zzx|郑州西|XPF|zhengzhouxi|zzx|2140@abq|阿巴嘎旗|AQC|abagaqi|abgq|2141@acb|阿城北|ABB|achengbei|acb|2142@aeb|阿尔山北|ARX|aershanbei|aesb|2143@alt|阿勒泰|AUR|aletai|alt|2144@are|安仁|ARG|anren|ar|2145@asx|安顺西|ASE|anshunxi|asx|2146@atx|安图西|AXL|antuxi|atx|2147@ayd|安阳东|ADF|anyangdong|ayd|2148@bba|博白|BBZ|bobai|bb|2149@bbu|八步|BBE|babu|bb|2150@bch|栟茶|FWH|bencha|bc|2151@bdd|保定东|BMP|baodingdong|bdd|2152@bfs|八方山|FGQ|bafangshan|bfs|2153@bgo|白沟|FEP|baigou|bg|2154@bha|滨海|FHP|binhai|bh|2155@bhb|滨海北|FCP|binhaibei|bhb|2156@bjn|宝鸡南|BBY|baojinan|bjn|2157@bjz|北井子|BRT|beijingzi|bjz|2158@bmj|白马井|BFQ|baimajing|bmj|2159@bqi|宝清|BUB|baoqing|bq|2160@bsh|璧山|FZW|bishan|bs|2161@bsp|白沙铺|BSN|baishapu|bsp|2162@bsx|白水县|BGY|baishuixian|bsx|2163@bta|板塘|NGQ|bantang|bt|2164@bwd|白文东|BCV|baiwendong|bwd|2165@bxb|宾西北|BBB|binxibei|bxb|2166@bxc|本溪新城|BVT|benxixincheng|bxxc|2167@bxi|彬县|BXY|binxian|bx|2168@bya|宾阳|UKZ|binyang|by|2169@byd|白洋淀|FWP|baiyangdian|byd|2170@byi|百宜|FHW|baiyi|by|2171@byn|白音华南|FNC|baiyinhuanan|byhn|2172@bzd|巴中东|BDE|bazhongdong|bzd|2173@bzh|滨州|BIK|binzhou|bz|2174@bzh|宾州|BZB|binzhou|bz|2175@bzx|霸州西|FOP|bazhouxi|bzx|2176@cch|澄城|CUY|chengcheng|cc|2177@cgb|城固北|CBY|chenggubei|cgb|2178@cgh|查干湖|VAT|chaganhu|cgh|2179@chd|巢湖东|GUH|chaohudong|chd|2180@cji|从江|KNW|congjiang|cj|2181@cjy|蔡家崖|EBV|caijiaya|cjy|2182@cka|茶卡|CVO|chaka|ck|2183@clh|长临河|FVH|changlinhe|clh|2184@cln|茶陵南|CNG|chalingnan|cln|2185@cpd|常平东|FQQ|changpingdong|cpd|2186@cpn|常平南|FPQ|changpingnan|cpn|2187@cqq|长庆桥|CQJ|changqingqiao|cqq|2188@csb|长寿北|COW|changshoubei|csb|2189@csh|长寿湖|CSE|changshouhu|csh|2190@csh|常山|CSU|changshan|cs|2191@csh|潮汕|CBQ|chaoshan|cs|2192@csx|长沙西|RXQ|changshaxi|csx|2193@cti|朝天|CTE|chaotian|ct|2194@ctn|长汀南|CNS|changtingnan|ctn|2195@cwu|长武|CWY|changwu|cw|2196@cxi|长兴|CBH|changxing|cx|2197@cxi|苍溪|CXE|cangxi|cx|2198@cxi|楚雄|CUM|chuxiong|cx|2199@cya|长阳|CYN|changyang|cy|2200@cya|潮阳|CNQ|chaoyang|cy|2201@czt|城子坦|CWT|chengzitan|czt|2202@dad|东安东|DCZ|dongandong|dad|2203@dba|德保|RBZ|debao|db|2204@dch|都昌|DCG|duchang|dc|2205@dch|东岔|DCJ|dongcha|dc|2206@dcn|东城南|IYQ|dongchengnan|dcn|2207@ddh|东戴河|RDD|dongdaihe|ddh|2208@ddx|丹东西|RWT|dandongxi|ddx|2209@deh|东二道河|DRB|dongerdaohe|dedh|2210@dfe|大丰|KRQ|dafeng|df|2211@dfn|大方南|DNE|dafangnan|dfn|2212@dgb|东港北|RGT|donggangbei|dgb|2213@dgs|大孤山|RMT|dagushan|dgs|2214@dgu|东莞|RTQ|dongguan|dg|2215@dhd|鼎湖东|UWQ|dinghudong|dhd|2216@dhs|鼎湖山|NVQ|dinghushan|dhs|2217@dji|道滘|RRQ|daojiao|dj|2218@dji|垫江|DJE|dianjiang|dj|2219@dji|洞井|FWQ|dongjing|dj|2220@dju|大苴|DIM|daju|dj|2221@dlh|达连河|DCB|dalianhe|dlh|2222@dli|大荔|DNY|dali|dl|2223@dlz|大朗镇|KOQ|dalangzhen|dlz|2224@dml|得莫利|DTB|demoli|dml|2225@dqg|大青沟|DSD|daqinggou|dqg|2226@dqi|德清|DRH|deqing|dq|2227@dsd|东胜东|RSC|dongshengdong|dsd|2228@dsn|砀山南|PRH|dangshannan|dsn|2229@dsn|大石头南|DAL|dashitounan|dstn|2230@dtd|当涂东|OWH|dangtudong|dtd|2231@dtx|大通西|DTO|datongxi|dtx|2232@dwa|大旺|WWQ|dawang|dw|2233@dxb|定西北|DNJ|dingxibei|dxb|2234@dxd|德兴东|DDG|dexingdong|dxd|2235@dxi|德兴|DWG|dexing|dx|2236@dxs|丹霞山|IRQ|danxiashan|dxs|2237@dyb|大冶北|DBN|dayebei|dyb|2238@dyd|都匀东|KJW|duyundong|dyd|2239@dyn|东营南|DOK|dongyingnan|dyn|2240@dyu|大余|DYG|dayu|dy|2241@dzd|定州东|DOP|dingzhoudong|dzd|2242@dzh|端州|WZQ|duanzhou|dz|2243@dzn|大足南|FQW|dazunan|dzn|2244@ems|峨眉山|IXW|emeishan|ems|2245@epg|阿房宫|EGY|epanggong|epg|2246@ezd|鄂州东|EFN|ezhoudong|ezd|2247@fcb|防城港北|FBZ|fangchenggangbei|fcgb|2248@fcd|凤城东|FDT|fengchengdong|fcd|2249@fch|富川|FDZ|fuchuan|fc|2250@fcx|繁昌西|PUH|fanchangxi|fcx|2251@fdu|丰都|FUW|fengdu|fd|2252@flb|涪陵北|FEW|fulingbei|flb|2253@fli|枫林|FLN|fenglin|fl|2254@fni|富宁|FNM|funing|fn|2255@fpi|佛坪|FUY|foping|fp|2256@fqi|法启|FQE|faqi|fq|2257@frn|芙蓉南|KCQ|furongnan|frn|2258@fsh|复盛|FAW|fusheng|fs|2259@fso|抚松|FSL|fusong|fs|2260@fsx|佛山西|FOQ|foshanxi|fsx|2261@fsz|福山镇|FZQ|fushanzhen|fsz|2262@fti|福田|NZQ|futian|ft|2263@fyb|富源北|FBM|fuyuanbei|fyb|2264@fyu|抚远|FYB|fuyuan|fy|2265@fzd|抚州东|FDG|fuzhoudong|fzd|2266@fzh|抚州|FZG|fuzhou|fz|2267@fzh|方正|FNB|fangzheng|fz|2268@gan|高安|GCG|gaoan|ga|2269@gan|广安南|VUW|guangannan|gan|2270@gan|贵安|GAE|guian|ga|2271@gbd|高碑店东|GMP|gaobeidiandong|gbdd|2272@gch|恭城|GCZ|gongcheng|gc|2273@gcn|藁城南|GUP|gaochengnan|gcn|2274@gdb|贵定北|FMW|guidingbei|gdb|2275@gdn|葛店南|GNN|gediannan|gdn|2276@gdx|贵定县|KIW|guidingxian|gdx|2277@ghb|广汉北|GVW|guanghanbei|ghb|2278@ghu|高花|HGD|gaohua|gh|2279@gju|革居|GEM|geju|gj|2280@gle|高楞|GLB|gaoleng|gl|2281@gli|关岭|GLE|guanling|gl|2282@glx|桂林西|GEZ|guilinxi|glx|2283@gmc|光明城|IMQ|guangmingcheng|gmc|2284@gni|广宁|FBQ|guangning|gn|2285@gns|广宁寺|GQT|guangningsi|gns|2286@gnx|广南县|GXM|guangnanxian|gnx|2287@gpi|桂平|GAZ|guiping|gp|2288@gpz|弓棚子|GPT|gongpengzi|gpz|2289@gsd|赶水东|GDE|ganshuidong|gsd|2290@gsh|光山|GUN|guangshan|gs|2291@gsh|谷山|FFQ|gushan|gs|2292@gsl|观沙岭|FKQ|guanshaling|gsl|2293@gtb|古田北|GBS|gutianbei|gtb|2294@gtb|广通北|GPM|guangtongbei|gtb|2295@gtn|高台南|GAJ|gaotainan|gtn|2296@gtz|古田会址|STS|gutianhuizhi|gthz|2297@gyb|贵阳北|KQW|guiyangbei|gyb|2298@gyd|贵阳东|KEW|guiyangdong|gyd|2299@gyx|高邑西|GNP|gaoyixi|gyx|2300@han|惠安|HNS|huian|ha|2301@hbb|淮北北|PLH|huaibeibei|hbb|2302@hbd|鹤壁东|HFF|hebidong|hbd|2303@hcg|寒葱沟|HKB|hanconggou|hcg|2304@hch|霍城|SER|huocheng|hc|2305@hch|珲春|HUL|hunchun|hc|2306@hdd|邯郸东|HPP|handandong|hdd|2307@hdd|横道河子东|KUX|hengdaohezidong|hdhzd|2308@hdo|惠东|KDQ|huidong|hd|2309@hdp|哈达铺|HDJ|hadapu|hdp|2310@hdx|洪洞西|HTV|hongtongxi|hdx|2311@hdx|海东西|HDO|haidongxi|hdx|2312@heb|哈尔滨北|HTB|haerbinbei|hebb|2313@hfc|合肥北城|COH|hefeibeicheng|hfbc|2314@hfn|合肥南|ENH|hefeinan|hfn|2315@hga|黄冈|KGN|huanggang|hg|2316@hgd|黄冈东|KAN|huanggangdong|hgd|2317@hgd|横沟桥东|HNN|henggouqiaodong|hgqd|2318@hgx|黄冈西|KXN|huanggangxi|hgx|2319@hhe|洪河|HPB|honghe|hh|2320@hhn|怀化南|KAQ|huaihuanan|hhn|2321@hhq|黄河景区|HCF|huanghejingqu|hhjq|2322@hhu|惠环|KHQ|huihuan|hh|2323@hhu|花湖|KHN|huahu|hh|2324@hhu|后湖|IHN|houhu|hh|2325@hji|怀集|FAQ|huaiji|hj|2326@hkb|河口北|HBM|hekoubei|hkb|2327@hkl|宏克力|OKB|hongkeli|hkl|2328@hlb|海林北|KBX|hailinbei|hlb|2329@hli|黄流|KLQ|huangliu|hl|2330@hln|黄陵南|VLY|huanglingnan|hln|2331@hme|鲘门|KMQ|houmen|hm|2332@hme|虎门|IUQ|humen|hm|2333@hmx|侯马西|HPV|houmaxi|hmx|2334@hna|衡南|HNG|hengnan|hn|2335@hnd|淮南东|HOH|huainandong|hnd|2336@hpu|合浦|HVZ|hepu|hp|2337@hqi|霍邱|FBH|huoqiu|hq|2338@hrd|怀仁东|HFV|huairendong|hrd|2339@hrd|华容东|HPN|huarongdong|hrd|2340@hrn|华容南|KRN|huarongnan|hrn|2341@hsb|衡水北|IHP|hengshuibei|hsb|2342@hsb|黄石北|KSN|huangshibei|hsb|2343@hsb|黄山北|NYH|huangshanbei|hsb|2344@hsd|贺胜桥东|HLN|heshengqiaodong|hsqd|2345@hsh|和硕|VUR|heshuo|hs|2346@hsn|花山南|KNN|huashannan|hsn|2347@hta|荷塘|KXQ|hetang|ht|2348@htd|黄土店|HKP|huangtudian|htd|2349@hyb|海阳北|HEK|haiyangbei|hyb|2350@hyb|合阳北|HTY|heyangbei|hyb|2351@hyi|槐荫|IYN|huaiyin|hy|2352@hyi|鄠邑|KXY|huyi|hyi|2353@hyk|花园口|HYT|huayuankou|hyk|2354@hzd|霍州东|HWV|huozhoudong|hzd|2355@hzn|惠州南|KNQ|huizhounan|hzn|2356@jan|建安|JUL|jianan|ja|2357@jch|泾川|JAJ|jingchuan|jc|2358@jdb|景德镇北|JDG|jingdezhenbei|jdzb|2359@jde|旌德|NSH|jingde|jd|2360@jfe|尖峰|PFQ|jianfeng|jf|2361@jha|近海|JHD|jinhai|jh|2362@jhx|蛟河西|JOL|jiaohexi|jhx|2363@jlb|军粮城北|JMP|junliangchengbei|jlcb|2364@jle|将乐|JLS|jiangle|jl|2365@jlh|贾鲁河|JLF|jialuhe|jlh|2366@jls|九郎山|KJQ|jiulangshan|jls|2367@jmb|即墨北|JVK|jimobei|jmb|2368@jmg|剑门关|JME|jianmenguan|jmg|2369@jmx|佳木斯西|JUB|jiamusixi|jmsx|2370@jnb|建宁县北|JCS|jianningxianbei|jnxb|2371@jni|江宁|JJH|jiangning|jn|2372@jnx|江宁西|OKH|jiangningxi|jnx|2373@jox|建瓯西|JUS|jianouxi|jox|2374@jqn|酒泉南|JNJ|jiuquannan|jqn|2375@jrx|句容西|JWH|jurongxi|jrx|2376@jsh|建水|JSM|jianshui|js|2377@jsh|尖山|JPQ|jianshan|js|2378@jss|界首市|JUN|jieshoushi|jss|2379@jxb|绩溪北|NRH|jixibei|jxb|2380@jxd|介休东|JDV|jiexiudong|jxd|2381@jxi|泾县|LOH|jingxian|jx|2382@jxi|靖西|JMZ|jingxi|jx|2383@jxn|进贤南|JXG|jinxiannan|jxn|2384@jyb|江油北|JBE|jiangyoubei|jyb|2385@jyn|简阳南|JOW|jianyangnan|jyn|2386@jyn|嘉峪关南|JBJ|jiayuguannan|jygn|2387@jyt|金银潭|JTN|jinyintan|jyt|2388@jyu|靖宇|JYL|jingyu|jy|2389@jyw|金月湾|PYQ|jinyuewan|jyw|2390@jyx|缙云西|PYH|jinyunxi|jyx|2391@jzh|景州|JEP|jingzhou|jz|2392@jzh|晋中|JZV|jinzhong|jz|2393@kfb|开封北|KBF|kaifengbei|kfb|2394@kfs|开福寺|FLQ|kaifusi|kfs|2395@khu|开化|KHU|kaihua|kh|2396@kln|凯里南|QKW|kailinan|kln|2397@klu|库伦|KLD|kulun|kl|2398@kmn|昆明南|KOM|kunmingnan|kmn|2399@kta|葵潭|KTQ|kuitan|kt|2400@kya|开阳|KVW|kaiyang|ky|2401@lad|隆安东|IDZ|longandong|lad|2402@lbb|来宾北|UCZ|laibinbei|lbb|2403@lbi|灵璧|GMH|lingbi|lb|2404@lbu|寮步|LTQ|liaobu|lb|2405@lby|绿博园|LCF|lvboyuan|lby|2406@lcb|隆昌北|NWW|longchangbei|lcb|2407@lcd|乐昌东|ILQ|lechangdong|lcd|2408@lch|临城|UUP|lincheng|lc|2409@lch|罗城|VCZ|luocheng|lc|2410@lch|陵城|LGK|lingcheng|lc|2411@lcz|老城镇|ACQ|laochengzhen|lcz|2412@ldb|龙洞堡|FVW|longdongbao|ldb|2413@ldn|乐都南|LVO|ledunan|ldn|2414@ldn|娄底南|UOQ|loudinan|ldn|2415@ldo|乐东|UQQ|ledong|ld|2416@ldy|离堆公园|INW|liduigongyuan|ldgy|2417@lfa|娄烦|USV|loufan|lf|2418@lfe|陆丰|LLQ|lufeng|lf|2419@lfe|龙丰|KFQ|longfeng|lf|2420@lfn|禄丰南|LQM|lufengnan|lfn|2421@lfx|临汾西|LXV|linfenxi|lfx|2422@lgn|临高南|KGQ|lingaonan|lgn|2423@lgu|麓谷|BNQ|lugu|lg|2424@lhe|滦河|UDP|luanhe|lh|2425@lhn|珞璜南|LNE|luohuangnan|lhn|2426@lhx|漯河西|LBN|luohexi|lhx|2427@ljd|罗江东|IKW|luojiangdong|ljd|2428@lji|柳江|UQZ|liujiang|lj|2429@ljn|利津南|LNK|lijinnan|ljn|2430@lkn|兰考南|LUF|lankaonan|lkn|2431@lks|龙口市|UKK|longkoushi|lks|2432@llb|龙里北|KFW|longlibei|llb|2433@llb|兰陵北|COK|lanlingbei|llb|2434@llb|沥林北|KBQ|lilinbei|llb|2435@lld|醴陵东|UKQ|lilingdong|lld|2436@lna|陇南|INJ|longnan|ln|2437@lpn|梁平南|LPE|liangpingnan|lpn|2438@lqu|礼泉|LGY|liquan|lq|2439@lsd|灵石东|UDV|lingshidong|lsd|2440@lsh|乐山|IVW|leshan|ls|2441@lsh|龙市|LAG|longshi|ls|2442@lsh|溧水|LDH|lishui|ls|2443@lsn|娄山关南|LSE|loushanguannan|lsgn|2444@lwj|洛湾三江|KRW|luowansanjiang|lwsj|2445@lxb|莱西北|LBK|laixibei|lxb|2446@lxi|岚县|UXV|lanxian|lx|2447@lya|溧阳|LEH|liyang|ly|2448@lyi|临邑|LUK|linyi|ly|2449@lyn|柳园南|LNR|liuyuannan|lyn|2450@lzb|鹿寨北|LSZ|luzhaibei|lzb|2451@lzh|阆中|LZE|langzhong|lz|2452@lzn|临泽南|LDJ|linzenan|lzn|2453@mad|马鞍山东|OMH|maanshandong|masd|2454@mch|毛陈|MHN|maochen|mc|2455@mex|帽儿山西|MUB|mershanxi|mesx|2456@mgd|明港东|MDN|minggangdong|mgd|2457@mhn|民和南|MNO|minhenan|mhn|2458@mji|闵集|MJN|minji|mj|2459@mla|马兰|MLR|malan|ml|2460@mle|民乐|MBJ|minle|ml|2461@mle|弥勒|MLM|mile|ml|2462@mns|玛纳斯|MSR|manasi|mns|2463@mpi|牟平|MBK|muping|mp|2464@mqb|闽清北|MBS|minqingbei|mqb|2465@mqb|民权北|MIF|minquanbei|mqb|2466@msd|眉山东|IUW|meishandong|msd|2467@msh|庙山|MSN|miaoshan|ms|2468@mxi|岷县|MXJ|minxian|mx|2469@myu|门源|MYO|menyuan|my|2470@myu|暮云|KIQ|muyun|my|2471@mzb|蒙自北|MBM|mengzibei|mzb|2472@mzh|孟庄|MZF|mengzhuang|mz|2473@mzi|蒙自|MZM|mengzi|mz|2474@nbu|南部|NBE|nanbu|nb|2475@nca|南曹|NEF|nancao|nc|2476@ncb|南充北|NCE|nanchongbei|ncb|2477@nch|南城|NDG|nancheng|nc|2478@nch|南 昌|NOG|nanchang|nc|2479@ncx|南昌西|NXG|nanchangxi|ncx|2480@ndn|宁东南|NDJ|ningdongnan|ndn|2481@ndo|宁东|NOJ|ningdong|nd|2482@nfb|南芬北|NUT|nanfenbei|nfb|2483@nfe|南丰|NFG|nanfeng|nf|2484@nhd|南湖东|NDN|nanhudong|nhd|2485@nhu|南华|NAM|nanhua|nh|2486@njb|内江北|NKW|neijiangbei|njb|2487@nji|南江|FIW|nanjiang|nj|2488@njk|南江口|NDQ|nanjiangkou|njk|2489@nli|南陵|LLH|nanling|nl|2490@nmu|尼木|NMO|nimu|nm|2491@nnd|南宁东|NFZ|nanningdong|nnd|2492@nnx|南宁西|NXZ|nanningxi|nnx|2493@npb|南平北|NBS|nanpingbei|npb|2494@nqn|宁强南|NOY|ningqiangnan|nqn|2495@nxi|南雄|NCQ|nanxiong|nx|2496@nyo|纳雍|NYE|nayong|ny|2497@nyz|南阳寨|NYF|nanyangzhai|nyz|2498@pan|普安|PAN|puan|pa|2499@pax|普安县|PUE|puanxian|pax|2500@pbi|屏边|PBM|pingbian|pb|2501@pbn|平坝南|PBE|pingbanan|pbn|2502@pch|平昌|PCE|pingchang|pc|2503@pdi|普定|PGW|puding|pd|2504@pdu|平度|PAK|pingdu|pd|2505@pko|皮口|PUT|pikou|pk|2506@plc|盘龙城|PNN|panlongcheng|plc|2507@pls|蓬莱市|POK|penglaishi|pls|2508@pni|普宁|PEQ|puning|pn|2509@pnn|平南南|PAZ|pingnannan|pnn|2510@psb|彭山北|PPW|pengshanbei|psb|2511@psh|盘山|PUD|panshan|ps|2512@psh|坪上|PSK|pingshang|ps|2513@pxb|萍乡北|PBG|pingxiangbei|pxb|2514@pya|鄱阳|PYG|poyang|py|2515@pya|濮阳|PYF|puyang|py|2516@pyc|平遥古城|PDV|pingyaogucheng|pygc|2517@pyd|平原东|PUK|pingyuandong|pyd|2518@pzh|盘州|PAE|panzhou|pz|2519@pzh|普者黑|PZM|puzhehei|pzh|2520@pzh|彭州|PMW|pengzhou|pz|2521@qan|秦安|QGJ|qinan|qa|2522@qbd|青白江东|QFW|qingbaijiangdong|qbjd|2523@qch|青川|QCE|qingchuan|qc|2524@qdb|青岛北|QHK|qingdaobei|qdb|2525@qdo|祁东|QMQ|qidong|qd|2526@qdu|青堆|QET|qingdui|qd|2527@qfe|前锋|QFB|qianfeng|qf|2528@qjb|曲靖北|QBM|qujingbei|qjb|2529@qjd|綦江东|QDE|qijiangdong|qjd|2530@qji|曲江|QIM|qujiang|qj|2531@qli|青莲|QEW|qinglian|ql|2532@qqn|齐齐哈尔南|QNB|qiqihaernan|qqhen|2533@qsb|清水北|QEJ|qingshuibei|qsb|2534@qsh|青神|QVW|qingshen|qs|2535@qsh|岐山|QAY|qishan|qs|2536@qsh|庆盛|QSQ|qingsheng|qs|2537@qsx|清水县|QIJ|qingshuixian|qsx|2538@qsx|曲水县|QSO|qushuixian|qsx|2539@qxd|祁县东|QGV|qixiandong|qxd|2540@qxi|乾县|QBY|qianxian|qx|2541@qxn|旗下营南|QNC|qixiayingnan|qxyn|2542@qya|祁阳|QWQ|qiyang|qy|2543@qzn|全州南|QNZ|quanzhounan|qzn|2544@qzw|棋子湾|QZQ|qiziwan|qzw|2545@rbu|仁布|RUO|renbu|rb|2546@rcb|荣昌北|RQW|rongchangbei|rcb|2547@rch|荣成|RCK|rongcheng|rc|2548@rcx|瑞昌西|RXG|ruichangxi|rcx|2549@rdo|如东|RIH|rudong|rd|2550@rji|榕江|RVW|rongjiang|rj|2551@rkz|日喀则|RKO|rikaze|rkz|2552@rpi|饶平|RVQ|raoping|rp|2553@scl|宋城路|SFF|songchenglu|scl|2554@sdh|三道湖|SDL|sandaohu|sdh|2555@sdo|邵东|FIQ|shaodong|sd|2556@sdx|三都县|KKW|sanduxian|sdx|2557@sfa|胜芳|SUP|shengfang|sf|2558@sfb|双峰北|NFQ|shuangfengbei|sfb|2559@she|商河|SOK|shanghe|sh|2560@sho|泗洪|GQH|sihong|sh|2561@shu|四会|AHQ|sihui|sh|2562@sjd|石家庄东|SXP|shijiazhuangdong|sjzd|2563@sjn|三江南|SWZ|sanjiangnan|sjn|2564@sjz|三井子|OJT|sanjingzi|sjz|2565@slc|双流机场|IPW|shuangliujichang|sljc|2566@slh|双龙湖|OHB|shuanglonghu|slh|2567@slx|石林西|SYM|shilinxi|slx|2568@slx|沙岭子西|IXP|shalingzixi|slzx|2569@slx|双流西|IQW|shuangliuxi|slx|2570@slz|胜利镇|OLB|shenglizhen|slz|2571@smb|三明北|SHS|sanmingbei|smb|2572@smi|嵩明|SVM|songming|sm|2573@sml|树木岭|FMQ|shumuling|sml|2574@smu|神木|HMY|shenmu|sm|2575@snq|苏尼特左旗|ONC|sunitezuoqi|sntzq|2576@spd|山坡东|SBN|shanpodong|spd|2577@sqi|石桥|SQE|shiqiao|sq|2578@sqi|沈丘|SQN|shenqiu|sq|2579@ssb|鄯善北|SMR|shanshanbei|ssb|2580@ssb|狮山北|NSQ|shishanbei|ssb|2581@ssb|三水北|ARQ|sanshuibei|ssb|2582@ssb|松山湖北|KUQ|songshanhubei|sshb|2583@ssh|狮山|KSQ|shishan|ss|2584@ssn|三水南|RNQ|sanshuinan|ssn|2585@ssn|韶山南|INQ|shaoshannan|ssn|2586@ssu|三穗|QHW|sansui|ss|2587@sti|石梯|STE|shiti|st|2588@swe|汕尾|OGQ|shanwei|sw|2589@sxb|歙县北|NPH|shexianbei|sxb|2590@sxb|绍兴北|SLH|shaoxingbei|sxb|2591@sxd|绍兴东|SSH|shaoxingdong|sxd|2592@sxi|泗县|GPH|sixian|sx|2593@sxi|始兴|IPQ|shixing|sx|2594@sya|泗阳|MPH|siyang|sy|2595@sya|双阳|OYT|shuangyang|sy|2596@syb|邵阳北|OVQ|shaoyangbei|syb|2597@syb|松原北|OCT|songyuanbei|syb|2598@syi|山阴|SNV|shanyin|sy|2599@szb|深圳北|IOQ|shenzhenbei|szb|2600@szh|神州|SRQ|shenzhou|sz|2601@szn|尚志南|OZB|shangzhinan|szn|2602@szs|深圳坪山|IFQ|shenzhenpingshan|szps|2603@szs|石嘴山|QQJ|shizuishan|szs|2604@szx|石柱县|OSW|shizhuxian|szx|2605@tan|台安南|TAD|taiannan|tan|2606@tcb|桃村北|TOK|taocunbei|tcb|2607@tdb|田东北|TBZ|tiandongbei|tdb|2608@tdd|土地堂东|TTN|tuditangdong|tdtd|2609@tgx|太谷西|TIV|taiguxi|tgx|2610@tha|吐哈|THR|tuha|th|2611@tha|通海|TAM|tonghai|th|2612@thb|太和北|JYN|taihebei|thb|2613@thc|天河机场|TJN|tianhejichang|thjc|2614@thj|天河街|TEN|tianhejie|thj|2615@thx|通化县|TXL|tonghuaxian|thx|2616@tji|同江|TJB|tongjiang|tj|2617@tkd|托克托东|TVC|tuoketuodong|tktd|2618@tlb|吐鲁番北|TAR|tulufanbei|tlfb|2619@tlb|铜陵北|KXH|tonglingbei|tlb|2620@tni|泰宁|TNS|taining|tn|2621@trn|铜仁南|TNW|tongrennan|trn|2622@tsn|天水南|TIJ|tianshuinan|tsn|2623@twe|通渭|TWJ|tongwei|tw|2624@txd|田心东|KQQ|tianxindong|txd|2625@txh|汤逊湖|THN|tangxunhu|txh|2626@txi|藤县|TAZ|tengxian|tx|2627@tyn|太原南|TNV|taiyuannan|tyn|2628@tyx|通远堡西|TST|tongyuanpuxi|typx|2629@tzb|桐梓北|TBE|tongzibei|tzb|2630@tzd|桐梓东|TDE|tongzidong|tzd|2631@tzh|通州|TOP|tongzhou|tz|2632@wch|吴川|WAQ|wuchuan|wc|2633@wdd|文登东|WGK|wendengdong|wdd|2634@wfs|五府山|WFG|wufushan|wfs|2635@whb|威虎岭北|WBL|weihulingbei|whlb|2636@whb|威海北|WHK|weihaibei|whb|2637@whx|苇河西|WIB|weihexi|whx|2638@wlb|乌兰察布|WPC|wulanchabu|wlcb|2639@wld|五龙背东|WMT|wulongbeidong|wlbd|2640@wln|乌龙泉南|WFN|wulongquannan|wlqn|2641@wns|五女山|WET|wunvshan|wns|2642@wsh|武胜|WSE|wusheng|ws|2643@wto|五通|WTZ|wutong|wt|2644@wwe|无为|IIH|wuwei|ww|2645@wws|瓦屋山|WAH|wawushan|wws|2646@wxx|闻喜西|WOV|wenxixi|wxx|2647@wyb|武义北|WDH|wuyibei|wyb|2648@wyb|武夷山北|WBS|wuyishanbei|wysb|2649@wyd|武夷山东|WCS|wuyishandong|wysd|2650@wyu|婺源|WYG|wuyuan|wy|2651@wyu|渭源|WEJ|weiyuan|wy|2652@wzb|万州北|WZE|wanzhoubei|wzb|2653@wzh|武陟|WIF|wuzhi|wz|2654@wzn|梧州南|WBZ|wuzhounan|wzn|2655@xab|兴安北|XDZ|xinganbei|xab|2656@xcd|许昌东|XVF|xuchangdong|xcd|2657@xch|项城|ERN|xiangcheng|xc|2658@xdd|新都东|EWW|xindudong|xdd|2659@xfe|西丰|XFT|xifeng|xf|2660@xfe|先锋|NQQ|xianfeng|xf|2661@xfl|湘府路|FVQ|xiangfulu|xfl|2662@xfx|襄汾西|XTV|xiangfenxi|xfx|2663@xgb|孝感北|XJN|xiaoganbei|xgb|2664@xgd|孝感东|GDN|xiaogandong|xgd|2665@xhd|西湖东|WDQ|xihudong|xhd|2666@xhn|新化南|EJQ|xinhuanan|xhn|2667@xhx|新晃西|EWQ|xinhuangxi|xhx|2668@xji|新津|IRW|xinjin|xj|2669@xjk|小金口|NKQ|xiaojinkou|xjk|2670@xjn|辛集南|IJP|xinjinan|xjn|2671@xjn|新津南|ITW|xinjinnan|xjn|2672@xnd|咸宁东|XKN|xianningdong|xnd|2673@xnn|咸宁南|UNN|xianningnan|xnn|2674@xpn|溆浦南|EMQ|xupunan|xpn|2675@xpx|西平西|EGQ|xipingxi|xpx|2676@xtb|湘潭北|EDQ|xiangtanbei|xtb|2677@xtd|邢台东|EDP|xingtaidong|xtd|2678@xwq|西乌旗|XWC|xiwuqi|xwq|2679@xwx|修武西|EXF|xiuwuxi|xwx|2680@xwx|修文县|XWE|xiuwenxian|xwx|2681@xxb|萧县北|QSH|xiaoxianbei|xxb|2682@xxb|新香坊北|RHB|xinxiangfangbei|xxfb|2683@xxd|新乡东|EGF|xinxiangdong|xxd|2684@xyb|新余北|XBG|xinyubei|xyb|2685@xyc|西阳村|XQF|xiyangcun|xyc|2686@xyd|信阳东|OYN|xinyangdong|xyd|2687@xyd|咸阳秦都|XOY|xianyangqindu|xyqd|2688@xyo|仙游|XWS|xianyou|xy|2689@xyu|祥云|XQM|xiangyun|xy|2690@xzc|新郑机场|EZF|xinzhengjichang|xzjc|2691@xzl|香樟路|FNQ|xiangzhanglu|xzl|2692@xzx|忻州西|IXV|xinzhouxi|xzx|2693@ybl|迎宾路|YFW|yingbinlu|ybl|2694@ybx|亚布力西|YSB|yabulixi|yblx|2695@ycb|永城北|RGH|yongchengbei|ycb|2696@ycb|运城北|ABV|yunchengbei|ycb|2697@ycd|永川东|WMW|yongchuandong|ycd|2698@ycd|禹城东|YSK|yuchengdong|ycd|2699@ych|宜春|YEG|yichun|yc|2700@ych|岳池|AWW|yuechi|yc|2701@ydh|云东海|NAQ|yundonghai|ydh|2702@ydu|姚渡|AOJ|yaodu|yd|2703@yfd|云浮东|IXQ|yunfudong|yfd|2704@yfn|永福南|YBZ|yongfunan|yfn|2705@yge|雨格|VTM|yuge|yg|2706@yhe|洋河|GTH|yanghe|yh|2707@yjb|永济北|AJV|yongjibei|yjb|2708@yji|弋江|RVH|yijiang|yj|2709@yjp|于家堡|YKP|yujiapu|yjp|2710@yjx|延吉西|YXL|yanjixi|yjx|2711@ykn|永康南|QUH|yongkangnan|ykn|2712@yla|依兰|YEB|yilan|yl|2713@ylh|运粮河|YEF|yunlianghe|ylh|2714@yli|炎陵|YAG|yanling|yl|2715@yln|杨陵南|YEY|yanglingnan|yln|2716@ymb|一面坡北|YXB|yimianpobei|ympb|2717@ymi|伊敏|YMX|yimin|ym|2718@yna|郁南|YKQ|yunan|yn|2719@yny|云南驿|ANM|yunnanyi|yny|2720@ypi|银瓶|KPQ|yinping|yp|2721@ypx|原平西|IPV|yuanpingxi|ypx|2722@yqx|阳曲西|IQV|yangquxi|yqx|2723@ysh|阳朔|YCZ|yangshuo|ys|2724@ysh|永寿|ASY|yongshou|ys|2725@ysh|云山|KZQ|yunshan|ys|2726@ysn|玉山南|YGG|yushannan|ysn|2727@yta|永泰|YTS|yongtai|yt|2728@yta|银滩|CTQ|yintan|yt|2729@ytb|鹰潭北|YKG|yingtanbei|ytb|2730@ytn|烟台南|YLK|yantainan|ytn|2731@yto|伊通|YTL|yitong|yt|2732@ytx|烟台西|YTK|yantaixi|ytx|2733@yxi|尤溪|YXS|youxi|yx|2734@yxi|云霄|YBS|yunxiao|yx|2735@yxi|宜兴|YUH|yixing|yx|2736@yxi|玉溪|AXM|yuxi|yx|2737@yxi|阳信|YVK|yangxin|yx|2738@yxi|应县|YZV|yingxian|yx|2739@yxn|攸县南|YXG|youxiannan|yxn|2740@yxx|洋县西|YXY|yangxianxi|yxx|2741@yxx|义县西|YSD|yixianxi|yxx|2742@yyb|余姚北|CTH|yuyaobei|yyb|2743@yzh|榆中|IZJ|yuzhong|yz|2744@zan|诏安|ZDS|zhaoan|za|2745@zdc|正定机场|ZHP|zhengdingjichang|zdjc|2746@zfd|纸坊东|ZMN|zhifangdong|zfd|2747@zge|准格尔|ZEC|zhungeer|zge|2748@zhb|庄河北|ZUT|zhuanghebei|zhb|2749@zhu|昭化|ZHW|zhaohua|zh|2750@zjb|织金北|ZJE|zhijinbei|zjb|2751@zjc|张家川|ZIJ|zhangjiachuan|zjc|2752@zji|芷江|ZPQ|zhijiang|zj|2753@zji|织金|IZW|zhijin|zj|2754@zka|仲恺|KKQ|zhongkai|zk|2755@zko|曾口|ZKE|zengkou|zk|2756@zli|珠琳|ZOM|zhulin|zl|2757@zli|左岭|ZSN|zuoling|zl|2758@zmd|樟木头东|ZRQ|zhangmutoudong|zmtd|2759@zmx|驻马店西|ZLN|zhumadianxi|zmdx|2760@zpu|漳浦|ZCS|zhangpu|zp|2761@zqd|肇庆东|FCQ|zhaoqingdong|zqd|2762@zqi|庄桥|ZQH|zhuangqiao|zq|2763@zsh|昭山|KWQ|zhaoshan|zs|2764@zsx|钟山西|ZAZ|zhongshanxi|zsx|2765@zxi|漳县|ZXJ|zhangxian|zx|2766@zyb|资阳北|FYW|ziyangbei|zyb|2767@zyi|遵义|ZYE|zunyi|zy|2768@zyn|遵义南|ZNE|zunyinan|zyn|2769@zyx|张掖西|ZEJ|zhangyexi|zyx|2770@zzb|资中北|WZW|zizhongbei|zzb|2771@zzd|涿州东|ZAP|zhuozhoudong|zzd|2772@zzd|枣庄东|ZNK|zaozhuangdong|zzd|2773@zzd|卓资东|ZDC|zhuozidong|zzd|2774@zzd|郑州东|ZAF|zhengzhoudong|zzd|2775@zzn|株洲南|KVQ|zhuzhounan|zzn|2776"
	var stationSlice = strings.Split(station_names, "@")
	for i, stationDetails := range stationSlice {
		if i == 0 {
			continue
		}
		tep := strings.Split(stationDetails, "|")
		stationMap[tep[1]] = tep[2]
	}
	//WZ:无座,F:动卧,M:一等座,O:二等座,1:硬座,3:硬卧,4:软卧,6:高级软卧,9:商务座
	seatTypeMap = map[string]string{
		"WZ": "无座",
		"F":  "动卧",
		"M":  "一等座",
		"O":  "二等座",
		"1":  "硬座",
		"3":  "硬卧",
		"4":  "软卧",
		"6":  "高级软卧",
		"9":  "商务座",
	}
	// 输入的座位类别映射为票详细信息的字段
	seatTicketMap = map[string]int{
		"WZ": 26,
		"F":  33,
		"M":  31,
		"O":  30,
		"1":  29,
		"3":  28,
		"4":  23,
		"6":  21,
		"9":  32,
	}

}
func getUrlRespHtml(strUrl string, postDict map[string]string) (bool, string) {
	var respHtml string = ""
	CurCookies = nil
	httpClient := &http.Client{
		Jar: gCurCookieJar,
	}
	var httpReq *http.Request
	if nil == postDict {
		httpReq, _ = http.NewRequest("GET", strUrl, nil)
	} else {
		postValues := url.Values{}
		for postKey, PostValue := range postDict {
			postValues.Set(postKey, PostValue)
		}
		postDataStr := postValues.Encode()
		postDataBytes := []byte(postDataStr)
		postBytesReader := bytes.NewReader(postDataBytes)
		httpReq, _ = http.NewRequest("POST", strUrl, postBytesReader)
		httpReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.84 Safari/537.36")
		httpReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range gCurCookies {
			httpReq.AddCookie(c)
		}
	}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {

		fmt.Print("Error 请检查网路\n  http get strUrl=%s response error=%s\n", strUrl, err.Error())
		return false, ""
	}
	defer httpResp.Body.Close()
	body, errReadAll := ioutil.ReadAll(httpResp.Body)
	if errReadAll != nil {
		fmt.Print("Error  :  get response for strUrl=%s got error=%s\n", strUrl, errReadAll.Error())
		return false, ""
	}
	gCurCookies = gCurCookieJar.Cookies(httpReq.URL)
	respHtml = string(body)
	return true, respHtml
}

/*
	/***********************
	验证码确认：
	  1、获取验证码：生成文件，保存在固定地址
	         输入：无
	         输出：成功生成就会返回true

*/

func getCode() (getCodeRe bool) {
	//request_url := "https://kyfw.12306.cn/passport/captcha/captcha-image?login_site=E&module=login&rand=sjrand&0.6758635422370105";
	request_url := captchaImageAPI
	postDict := map[string]string{}
	requestBool, loginPic := getUrlRespHtml(request_url, postDict)
	if requestBool == true {
		f, _ := os.Create("/Users/drzhang/loginPic.png") //创建文件
		_, err := io.WriteString(f, loginPic)
		defer f.Close()
		if err != nil {
			panic("读取验证码错误，重新生成")
			return false
		} else {
			fmt.Print("0：正确生成验证码\n")
			return true
		}
	} else {
		return false
	}

}

/*
	  2、验证验证码：
			 输入：验证码坐标（人眼识别）
	         输出：成功放回true
*/
func checkCode(code string) (chkCodeRe bool) {
	//fmt.Print(code)
	//request_url := "https://kyfw.12306.cn/passport/captcha/captcha-check";
	request_url := captchaCheckAPI
	postDict := map[string]string{
		"answer":     code,
		"login_site": "E",
		"rand":       "sjrand",
	}
	_, html := getUrlRespHtml(request_url, postDict)
	//fmt.Print(html)
	var dat map[string]string
	json.Unmarshal([]byte(html), &dat)
	if dat["result_code"] == "4" {
		fmt.Println("1：验证码校验通过")
		return true
	}
	fmt.Println("1：验证码校验失败")
	return false
}

/***********************
登陆
   输入：用户名、密码
   输出：成功标志，用户中文名
***********************/

func Login(username string, password string) (isLogin bool) {
	//request_url := "https://kyfw.12306.cn/passport/web/login"
	request_url := loginAPI
	postDict := map[string]string{
		"username": username,
		"password": password,
		"appid":    "otn",
	}
	_, html := getUrlRespHtml(request_url, postDict)
	var dat map[string]string
	json.Unmarshal([]byte(html), &dat)
	if dat["result_message"] == "登录成功" {
		fmt.Println("2：用户名密码验证通过")
		//request_url = "https://kyfw.12306.cn/passport/web/auth/uamtk"
		request_url = uamtkAPI
		postDict = map[string]string{
			"appid": "otn",
		}
		_, html := getUrlRespHtml(request_url, postDict)
		var dat2 map[string]interface{}
		json.Unmarshal([]byte(html), &dat2)
		newa := dat2["newapptk"].(string)
		if dat2["result_message"].(string) == "验证通过" {
			fmt.Println("3：uamtk验证通过")
			//request_url = "https://kyfw.12306.cn/otn/uamauthclient"
			request_url = uamauthclientAPI
			postDict = map[string]string{
				"tk": newa,
			}
			_, html = getUrlRespHtml(request_url, postDict)
			//fmt.Println(html)
			json.Unmarshal([]byte(html), &dat2)
			fmt.Println("4：uamauthclient验证通过")
			fmt.Printf("欢迎你，%s \n", dat2["username"].(string))
			//return true,dat2["username"].(string)
			return true
		}
	} else {
		//return false,""
		fmt.Println(html)

	}
	return false
}

/*
查询出来火车票有3中主要状态，由第2与第11个字段确定： 第2个字段有3中状态： 1： Y ：存在某种余票。 N：没有任何票的余票了（第二个字段为预定，但是不显示，且没有余额编码）。      IS_TIME_NOT_BUY  需要读取第二字段，获取预售时间
查询火车票
   输入：列车日期、出发站编码、到达站编码
   输出：列车详细信息
*/
func spinner(ctx context.Context) {
	for {
		select {
		default:
			for _, r := range `-\|/` {
				fmt.Printf("\r%c", r)
				time.Sleep(100 * time.Millisecond)
			}

		case <-ctx.Done():
			fmt.Println(ctx.Err())
			fmt.Println("spinner 停止了")
			return
		}
	}
}

func queryTrain(dateInput string, fromStationInput string, toStationInput string) (getTickInfo bool, tickInfo []string) {
	//buydate := "2018-12-22"
	type TickInfo struct {
		Flag   string
		Map    map[string]string
		Result []string
	}
	type TranInfo struct {
		Data       *TickInfo
		HttpStatus int
		Message    string
		Status     bool
	}
	var traninfo TranInfo
	//fmt.Println("*%")
	request_url := queryTicketAPI + "leftTicketDTO.train_date=" + dateInput + "&leftTicketDTO.from_station=" + stationMap[fromStationInput] + "&leftTicketDTO.to_station=" + stationMap[toStationInput] + "&purpose_codes=ADULT"
	_, html := getUrlRespHtml(request_url, nil)
	fmt.Println(html)
	/*
	 判断如果是无效查询，比如日期不对，地点不对，直接返回查询失败。主要查找 返回的字符串中存在“err_bot”
	*/
	if strings.Contains(html, "ERROR") == true {
		fmt.Println("服务器返回防止刷票等错误")
		fmt.Println(html)
		return false, nil
	}

	if strings.Contains(html, "err_bot") == true {
		fmt.Println("未查到有效信息，请检查日期、地点等")
		return false, nil
	}
	if err := json.Unmarshal([]byte(html), &traninfo); err != nil {
		fmt.Println(err)
	}
	//fmt.Println(traninfo.Data.Result)
	if traninfo.HttpStatus == 200 {

		return true, traninfo.Data.Result
	} else {
		return false, nil
	}
}

/***********************
 判断用户是否在铁总系统在线：
	输入：无
	输出：无
    ***********************/
func chkOnline() (isChkOn bool) {
	//request_url := "https://kyfw.12306.cn/otn/login/checkUser"
	request_url := checkUserAPI
	postDict := map[string]string{
		"_json_att": "",
	}
	_, html := getUrlRespHtml(request_url, postDict)
	//fmt.Println(html)
	//{"validateMessagesShowId":"_validatorMessage","status":true,"httpstatus":200,"data":{"flag":true},"messages":[],"validateMessages":{}}
	type chkUsrOnline struct {
		ValidateMessagesShowId string            `json:"validateMessagesShowId"`
		Status                 bool              `json:"status"`
		HttpStatus             int               `json:"httpstatus"`
		Data                   map[string]bool   `json:"data"`
		Messages               []string          `json:"messages"`
		ValidateMessages       map[string]string `json:"validateMessages"`
	}
	var chkuserOnline chkUsrOnline
	//var dat map[string]string
	json.Unmarshal([]byte(html), &chkuserOnline)
	if chkuserOnline.Status == true {
		return true
	} else {

		return false
	}
}

/**
	总输入： 选择的列车信息、购买日期、出发到达站点、座位类别


   buyTicket : 正式访问铁总数据库进行购票
   1、确认余票信息正确：   /otn/leftTicket/submitOrderRequest
         输入： 余票编码、出发日期、出发站点、到达站点
         输出： 确定余票信息可用
   2、分配用户提交码  /otn/confirmPassenger/initDc
         输入：无
         输出：globalRepeatSubmitToken，key_check_isChange
   3、获取用户信息，得到已有的乘客信息    /otn/confirmPassenger/getPassengerDTOs
         输入：globalRepeatSubmitToken
         输出：获取购票用户信息
   4、锁定票源 ：    /otn/confirmPassenger/checkOrderInfo
		输入：globalRepeatSubmitToken   购票用户信息  座位类别
        输出：无
   5、获取队列排队：  /otn/confirmPassenger/getQueueCount
        输入：globalRepeatSubmitToken   座位、购票用户信息  出发列车等
        输出：无
   6、购买票：  /otn/confirmPassenger/confirmSingleForQueue
        输入：globalRepeatSubmitToken，key_check_isChange，列车等各种信息（座位、出发、到达等）
        输出：最终购票成功
**/

func buyTicket(buyTicketName string, TickDetail string, dateInput string, fromStationInput string, toStationInput string, seatType string) (buyOK bool) {
	type chkUsrOnline struct {
		ValidateMessagesShowId string            `json:"validateMessagesShowId"`
		Status                 bool              `json:"status"`
		HttpStatus             int               `json:"httpstatus"`
		Data                   map[string]bool   `json:"data"`
		Messages               []string          `json:"messages"`
		ValidateMessages       map[string]string `json:"validateMessages"`
	}
	var chkuserOnline chkUsrOnline
	res2 := strings.Split(TickDetail, "|")
	//fmt.Println(res2)   //最终列表
	secretStr := res2[0]
	trainName := res2[3]
	trainNumber := res2[2]
	fromTelecode := res2[6]
	toTelecode := res2[7]
	startTime := res2[8]
	arriveTime := res2[9]
	travlLong := res2[10]
	leftTicket := res2[12]
	//trainDate:=res2[13]
	trainLocation := res2[15]
	//softBeach := res2[23]
	//hardBeach := res2[28]
	fmt.Println("购买车票信息如下：", trainName, " 出发时间：", startTime, "到达时间：", arriveTime, "历时：", travlLong, "座位类型:", seatTypeMap[seatType])
	//request_url := "https://kyfw.12306.cn/otn/leftTicket/submitOrderRequest"
	request_url := submitOrderRequestAPI
	v := "a=" + secretStr
	m, _ := url.ParseQuery(v)
	postDict := map[string]string{
		"secretStr":               m["a"][0],
		"train_date":              dateInput,
		"back_train_date":         time.Now().Format("2006-01-02"),
		"tour_flag":               "dc",
		"purpose_codes":           "ADULT",
		"query_from_station_name": fromStationInput,
		"query_to_station_name":   toStationInput,
		"undefined":               "",
	}
	_, html := getUrlRespHtml(request_url, postDict)
	json.Unmarshal([]byte(html), &chkuserOnline)
	if chkuserOnline.Status == true {
		fmt.Println("6：访问铁总后，用户可以购买该列车车票")
		//request_url = "https://kyfw.12306.cn/otn/confirmPassenger/initDc"
		request_url = initDcAPI
		postDict = map[string]string{
			"_json_att": "",
		}
		_, html = getUrlRespHtml(request_url, postDict)
		//fmt.Println(html)
		a := strings.Split(html, "var globalRepeatSubmitToken = '")
		c := strings.Split(html, "key_check_isChange':'")
		d := strings.Split(c[1], "','")
		//	fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n***********************************")
		//	fmt.Println(a[1])
		b := strings.Split(a[1], "';")
		//	fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n***********************************")
		//fmt.Println(b[0])
		//fmt.Println(d[0])
		//request_url = "https://kyfw.12306.cn/otn/confirmPassenger/getPassengerDTOs"
		request_url = getPassengerDTOsAPI
		postDict = map[string]string{
			"_json_att":           "",
			"REPEAT_SUBMIT_TOKEN": b[0],
		}
		_, html = getUrlRespHtml(request_url, postDict)

		type passengerDetail struct {
			Code                   string `json:"code"`
			Passenger_name         string `json:"passenger_name"`
			Sex_code               string `json:"sex_code"`
			Sex_name               string `json:"sex_name"`
			Born_date              string `json:"born_date"`
			Country_code           string `json:"country_code"`
			Passenger_id_type_code string `json:"passenger_id_type_code"`
			Passenger_id_type_name string `json:"passenger_id_type_name"`
			Passenger_id_no        string `json:"passenger_id_no"`
			Passenger_type         string `json:"passenger_type"`
			Passenger_flag         string `json:"passenger_flag"`
			Passenger_type_name    string `json:"passenger_type_name"`
			Mobile_no              string `json:"mobile_no"`
			Phone_no               string `json:"phone_no"`
			Email                  string `json:"email"`
			Address                string `json:"address"`
			Postalcode             string `json:"postalcode"`
			First_letter           string `json:"first_letter"`
			RecordCount            string `json:"recordCount"`
			Total_times            string `json:"total_times"`
			Index_id               string `json:"index_id"`
			Gat_born_date          string `json:"gat_born_date"`
			Gat_valid_date_start   string `json:"gat_valid_date_start"`
			Gat_valid_date_end     string `json:"gat_valid_date_end"`
			Gat_version            string `json:"gat_version"`
		}
		type passengerData struct {
			IsExist           bool               `json:"isExist"`
			ExMsg             string             `json:"exMsg"`
			Two_isOpenClick   []string           `json:"two_isOpenClick"`
			Other_isOpenClick []string           `json:"other_isOpenClick"`
			Normal_passengers []*passengerDetail `json:"normal_passengers"`
			Dj_passengers     []string           `json:"dj_passengers"`
		}
		type getPassengermessage struct {
			ValidateMessagesShowId string            `json:"validateMessagesShowId"`
			Status                 bool              `json:"status"`
			Httpstatus             int               `json:"httpstatus"`
			Data                   passengerData     `json:"data"`
			Messages               []string          `json:"messages"`
			ValidateMessages       map[string]string `json:"validateMessages"`
		}
		var getpassengermessage getPassengermessage
		if err := json.Unmarshal([]byte(html), &getpassengermessage); err != nil {
			fmt.Println(html)
			fmt.Println(err)
		}
		//fmt.Println("checkOrderInfo")
		//fmt.Println("获取已存用户信息，目前选择第一个用户")
		var i int = 0
		var tindex int = 0
		var s *passengerDetail
		for i, s = range getpassengermessage.Data.Normal_passengers {
			if s.Passenger_name == buyTicketName {
				tindex = i
				break
			}
		}
		fmt.Println("按照您的输入，购买 ", buyTicketName, "的车票，tindex为", tindex)
		var passengerTicketStr, oldPassengerStr string
		//fmt.Println(getpassengermessage.Data)
		passengerTicketStr = seatType + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_flag + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_type + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_name + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_id_type_code + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_id_no + "," + getpassengermessage.Data.Normal_passengers[tindex].Mobile_no + ",N"
		oldPassengerStr = getpassengermessage.Data.Normal_passengers[tindex].Passenger_name + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_id_type_code + "," + getpassengermessage.Data.Normal_passengers[tindex].Passenger_id_no + ",1_"
		//fmt.Println(passengerTicketStr)
		//fmt.Println(oldPassengerStr)
		//request_url = "https://kyfw.12306.cn/otn/confirmPassenger/checkOrderInfo"
		request_url = checkOrderInfoAPI
		postDict = map[string]string{
			"_json_att":           "",
			"bed_level_order_num": "000000000000000000000000000000",
			"cancel_flag":         "2",
			"oldPassengerStr":     oldPassengerStr,
			"passengerTicketStr":  passengerTicketStr, //包含了 座位类别
			"randCode":            "",
			"REPEAT_SUBMIT_TOKEN": b[0],
			"tour_flag":           "dc",
			"whatsSelect":         "1",
		}
		_, html = getUrlRespHtml(request_url, postDict)
		// 12月29日，增加了异常，铁总会在重复查询是返回错误
		if strings.Contains(html, "err_bot") == true {
			fmt.Println(html)
			fmt.Println("报错，网络存在问题？？")
			return false
		}
		//{"validateMessagesShowId":"_validatorMessage","status":true,"httpstatus":200,"data":{"ifShowPassCode":"N","canChooseBeds":"N","canChooseSeats":"Y","choose_Seats":"OM","isCanChooseMid":"N","ifShowPassCodeTime":"1145","submitStatus":true,"smokeStr":""},"messages":[],"validateMessages":{}}
		type dataDetails struct {
			IfShowPassCode     string `json:"ifShowPassCode"`
			CanChooseSeats     string `json:"canChooseSeats"`
			ErrMsg             string `json:"errMsg"`
			Choose_Seats       string `json:"choose_Seats"`
			IsCanChooseMid     string `json:"isCanChooseMid"`
			IfShowPassCodeTime string `json:"ifShowPassCodeTime"`
			SubmitStatus       bool   `json:"submitStatus"`
			SmokeStr           string `json:"smokeStr"`
		}

		type chkOrderSuccess struct {
			ValidateMessagesShowId string            `json:"validateMessagesShowId"`
			Status                 bool              `json:"status"`
			HttpStatus             int               `json:"httpstatus"`
			Data                   *dataDetails      `json:"data"`
			Messages               []string          `json:"messages"`
			ValidateMessages       map[string]string `json:"validateMessages"`
		}
		var chkordersuccess chkOrderSuccess
		if err := json.Unmarshal([]byte(html), &chkordersuccess); err != nil {
			fmt.Println(err)
		}
		if chkordersuccess.Data.SubmitStatus == true {
			fmt.Println("已锁定票源： 系统校验订单信息成功")
			dateSlice := strings.Split(dateInput, "-")
			yearInt, _ := strconv.Atoi(dateSlice[0])
			monthInt, _ := strconv.Atoi(dateSlice[1])
			dayInt, _ := strconv.Atoi(dateSlice[2])
			the_time := time.Date(yearInt, time.Month(monthInt), dayInt, 0, 0, 0, 0, time.Local)
			//request_url = "https://kyfw.12306.cn/otn/confirmPassenger/getQueueCount"
			request_url = getQueueCountAPI
			postDict = map[string]string{
				"_json_att":           "",
				"fromStationTelecode": fromTelecode,
				"leftTicket":          leftTicket,
				"purpose_codes":       "00",
				"REPEAT_SUBMIT_TOKEN": b[0],
				"seatType":            seatType,
				"stationTrainCode":    trainName,
				"toStationTelecode":   toTelecode,
				"train_date":          the_time.Format("Mon Jan 02 2006") + " 00: 00:00 GMT + 0800 (China Standard Time)",
				"train_location":      trainLocation,
				"train_no":            trainNumber,
			}
			_, html = getUrlRespHtml(request_url, postDict)
			//fmt.Println(html)
			type queueDetails struct {
				Count  string `json:"count"`
				Ticket string `json:"ticket"`
				Op_2   string `json:"op_2"`
				CountT string `json:"countT"`
				Op_1   string `json:"op_1"`
			}
			type chkQueueSuccess struct {
				ValidateMessagesShowId string            `json:"validateMessagesShowId"`
				Status                 bool              `json:"status"`
				HttpStatus             int               `json:"httpstatus"`
				Data                   *queueDetails     `json:"data"`
				Messages               []string          `json:"messages"`
				ValidateMessages       map[string]string `json:"validateMessages"`
			}
			var chkqueuesuccess chkQueueSuccess
			if err := json.Unmarshal([]byte(html), &chkqueuesuccess); err != nil {
				fmt.Println(err)
			}
			if chkqueuesuccess.Status == true {
				fmt.Println("系统获取队列信息成功", "counT:", chkqueuesuccess.Data.CountT, "count:", chkqueuesuccess.Data.Count, "Ticket:", chkqueuesuccess.Data.Ticket)
				//	fmt.Println("只差最后一步了，I will return")
				//    return true
				//request_url = "https://kyfw.12306.cn/otn/confirmPassenger/confirmSingleForQueue"
				request_url = confirmSingleForQueueAPI
				postDict = map[string]string{
					"passengerTicketStr":  passengerTicketStr,
					"oldPassengerStr":     oldPassengerStr,
					"randCode":            "",
					"purpose_codes":       "00",
					"key_check_isChange":  d[0],
					"leftTicketStr":       leftTicket,
					"train_location":      trainLocation,
					"choose_seats":        "", // 座位编号，如果为空，表示随机选择  xA ~~ xF
					"seatDetailType":      "000",
					"whatsSelect":         "1",
					"roomType":            "00",
					"dwAll":               "N",
					"_json_att":           "",
					"REPEAT_SUBMIT_TOKEN": b[0],
				}
				_, html = getUrlRespHtml(request_url, postDict)
				//fmt.Println(html)
				var configQueueSuccess chkOrderSuccess
				if err := json.Unmarshal([]byte(html), &configQueueSuccess); err != nil {
					fmt.Println(err)
				}
				if configQueueSuccess.Data.SubmitStatus == true {
					fmt.Println("确认队列 正式开始进行最终出票 \n   查看12306状态后回车")
					//var closeInput string
					//_, _ = fmt.Scanln(&closeInput)
					//循环开始获得 orderid
					request_url = queryOrderWaitTimeAPI
					postDict = map[string]string{
						"REPEAT_SUBMIT_TOKEN": b[0],
						"_json_att":           "",
						"tourFlag":            "dc",
						"random":              strconv.FormatInt(time.Now().Unix(), 10),
					}
					type orderDetails struct {
						Count                    int    `json:"count"`
						OrderId                  string `json:"orderId"`
						WaitTime                 int    `json:"waitTime"`
						WaitCount                int    `json:"waitCount"`
						QueryOrderWaitTimeStatus bool   `json:"queryOrderWaitTimeStatus"`
						Msg                      string `json:"msg"`
					}
					type chkOrderidSuccess struct {
						ValidateMessagesShowId string            `json:"validateMessagesShowId"`
						Status                 bool              `json:"status"`
						HttpStatus             int               `json:"httpstatus"`
						Data                   *orderDetails     `json:"data"`
						Messages               []string          `json:"messages"`
						ValidateMessages       map[string]string `json:"validateMessages"`
					}
					var chkorderidsuccess chkOrderidSuccess
					for {
						_, html = getUrlRespHtml(request_url, postDict)
						fmt.Println("获取最终订单号 ", html)
						if err := json.Unmarshal([]byte(html), &chkorderidsuccess); err != nil {
							fmt.Println(err)
							return false
						}
						if chkorderidsuccess.Data.QueryOrderWaitTimeStatus == true {
							if chkorderidsuccess.Data.OrderId != "" {
								//_, _ = fmt.Scanln(&closeInput)
								fmt.Println("排队获取了订单编号：", chkorderidsuccess.Data.OrderId)
								break
							} else if chkorderidsuccess.Data.Msg != "" {
								fmt.Println("排队获取订单号失败，现在处理掉失败订单，queryOrderWaitTimeAPI", chkorderidsuccess.Data.Msg)
								request_url2 := "https://kyfw.12306.cn/otn/queryOrder/queryMyOrderNoComplete"
								postDict2 := map[string]string{
									"_json_att": "",
								}
								_, html = getUrlRespHtml(request_url2, postDict2)
								fmt.Println(html)
								var waitInput string
								_, _ = fmt.Scanln(&waitInput)

								return false
							}
						} else {
							fmt.Println("购票失败：排队获取订单号失败，重新购买，queryOrderWaitTimeAPI")
							return false
						}
						//time.Sleep(50*time.Millisecond)
					}
					request_url = resultOrderForDcQueueAPI
					postDict = map[string]string{
						"REPEAT_SUBMIT_TOKEN": b[0],
						"_json_att":           "",
						"orderSequence_no":    chkorderidsuccess.Data.OrderId,
					}
					type buyOKDetails struct {
						SubmitStatue bool `json:"submitStatus"`
					}
					type chkbuyOKSuccess struct {
						ValidateMessagesShowId string            `json:"validateMessagesShowId"`
						Status                 bool              `json:"status"`
						HttpStatus             int               `json:"httpstatus"`
						Data                   *buyOKDetails     `json:"data"`
						Messages               []string          `json:"messages"`
						ValidateMessages       map[string]string `json:"validateMessages"`
					}
					var chkoksuccess chkbuyOKSuccess
					_, html = getUrlRespHtml(request_url, postDict)
					fmt.Println("最终确认出票 ", html)
					if err := json.Unmarshal([]byte(html), &chkoksuccess); err != nil {
						fmt.Println(err)
						return false
					}
					if chkoksuccess.Data.SubmitStatue == true {
						fmt.Println("购票出票，请前往12306支付\n" +
							"现在您铁总系统里包含如下车票，等3秒后反馈：")
						request_url2 := "https://kyfw.12306.cn/otn/queryOrder/queryMyOrderNoComplete"
						postDict2 := map[string]string{
							"_json_att": "",
						}
						time.Sleep(3 * time.Second)
						_, html = getUrlRespHtml(request_url2, postDict2)
						fmt.Println(html)

						type stationDTODeatil struct {
							Station_train_code string `json:"station_train_code"`
							From_station_name  string `json:"from_station_name"`
							To_station_name    string `json:"to_station_name"`
						}
						type passengerDTODeatil struct {
							Passenger_name string `json:"passenger_name"`
						}

						type ticketDetails struct {
							Ticket_status_name string             `json:"ticket_status_name"`
							Sequence_no        string             `json:"sequence_no"`
							StationTrainDTO    stationDTODeatil   `json:"stationTrainDTO"`
							PassengerDTO       passengerDTODeatil `json:"passengerDTO"`
						}
						type orderdbDetails struct {
							Sequence_no     string          `json:"sequence_no"`
							Order_date      string          `json:"order_date"`
							Ticket_totalnum int             `json:"ticket_totalnum"`
							Cancel_flag     string          `json:"cancel_flag"`
							Pay_flag        string          `json:"pay_flag"`
							Tickets         []ticketDetails `json:"tickets"`
						}

						type myorderDetails struct {
							OrderDBList []orderdbDetails `json:"orderDBlist"`
						}
						type myOrderInfo struct {
							ValidateMessagesShowId string            `json:"validateMessagesShowId"`
							Status                 bool              `json:"status"`
							HttpStatus             int               `json:"httpstatus"`
							Data                   myorderDetails    `json:"data"`
							Messages               []string          `json:"messages"`
							ValidateMessages       map[string]string `json:"validateMessages"`
						}
						var myorderinfo myOrderInfo
						if err := json.Unmarshal([]byte(html), &myorderinfo); err != nil {
							fmt.Print(html)
							fmt.Println(err)
						} else {
							var orderdbdetail orderdbDetails
							//fmt.Println("hello")
							if myorderinfo.Data.OrderDBList != nil {
								fmt.Println("您的订单如下：")
								// if myorderinfo.Data!=myorderDetails.nil{
								for _, orderdbdetail = range myorderinfo.Data.OrderDBList {
									fmt.Println("订单编号 ", orderdbdetail.Sequence_no, orderdbdetail.Cancel_flag, "乘车人 ", orderdbdetail.Tickets[0].PassengerDTO.Passenger_name,
										"列车编号 ", orderdbdetail.Tickets[0].StationTrainDTO.Station_train_code, "订单状态 ", orderdbdetail.Tickets[0].Ticket_status_name)
								}
							}
						}
						return true
					} else {
						fmt.Println("购票失败，resultOrderForDcQueueAPI")
						return false

					}

				} else {
					fmt.Println("configQueueSuccess  失败！！", configQueueSuccess.Data.ErrMsg)
					return false
				}

			} else {
				fmt.Println("系统获取队列信息失败")
				return false
			}
		} else {
			fmt.Println("未锁定票源：otn/confirmPassenger/checkOrderInfo 系统校验订单信息失败", chkordersuccess.Data.ErrMsg)
			return false
		}
	} else {
		fmt.Println("6：访问铁总后，该订单提交失败：", chkuserOnline.Messages)
		fmt.Println(html)
		return false
	}

}

// 根据车次座位查询票

func routineQueryTick(dateInput string, fromStationInput string, toStationInput string, trainName string, seatType string) (getTrainBoolRe bool, trainInfoDetailsRe string) {
	//var searchTick = false
	for {
		searchTickTimes++
		//200 毫秒会报 403 错误
		time.Sleep(300 * time.Millisecond)
		getTrainBool, trainInfoDetails := queryTrain(dateInput, fromStationInput, toStationInput)
		if getTrainBool == true {
			for _, trainDetail := range trainInfoDetails {
				//	fmt.Println(trainDetail)
				res := strings.Split(trainDetail, "|")
				//如果无此类型票，则显示“--”
				if res[3] != trainName {
					continue
				} else {
					if res[11] == "Y" && res[seatTicketMap[seatType]] != "无" {
						return getTrainBool, trainDetail
					} else {
						break
					}
				}
			}
			getTrainBool = false
		}

	}
	//return false,nil
}

func queryPassager() (int, []string) {
	type passengerDetail struct {
		Code                   string `json:"code"`
		Passenger_name         string `json:"passenger_name"`
		Sex_code               string `json:"sex_code"`
		Sex_name               string `json:"sex_name"`
		Born_date              string `json:"born_date"`
		Country_code           string `json:"country_code"`
		Passenger_id_type_code string `json:"passenger_id_type_code"`
		Passenger_id_type_name string `json:"passenger_id_type_name"`
		Passenger_id_no        string `json:"passenger_id_no"`
		Passenger_type         string `json:"passenger_type"`
		Passenger_flag         string `json:"passenger_flag"`
		Passenger_type_name    string `json:"passenger_type_name"`
		Mobile_no              string `json:"mobile_no"`
		Phone_no               string `json:"phone_no"`
		Email                  string `json:"email"`
		Address                string `json:"address"`
		Postalcode             string `json:"postalcode"`
		First_letter           string `json:"first_letter"`
		RecordCount            string `json:"recordCount"`
		Total_times            string `json:"total_times"`
		Index_id               string `json:"index_id"`
		Gat_born_date          string `json:"gat_born_date"`
		Gat_valid_date_start   string `json:"gat_valid_date_start"`
		Gat_valid_date_end     string `json:"gat_valid_date_end"`
		Gat_version            string `json:"gat_version"`
	}
	type passengerData struct {
		PageTotal int                `json:"pageTotal"`
		Datas     []*passengerDetail `json:"datas"`
		Flag      bool               `json:"flag"`
	}
	type getPassengermessage struct {
		ValidateMessagesShowId string            `json:"validateMessagesShowId"`
		Status                 bool              `json:"status"`
		Httpstatus             int               `json:"httpstatus"`
		Data                   passengerData     `json:"data"`
		Messages               []string          `json:"messages"`
		ValidateMessages       map[string]string `json:"validateMessages"`
	}
	var getpassengermessage getPassengermessage
	request_url := querypassagerAPI
	postDict := map[string]string{
		"pageIndex": "1",
		"pageSize":  "10",
	}
	_, html := getUrlRespHtml(request_url, postDict)
	//fmt.Println(html)
	if err := json.Unmarshal([]byte(html), &getpassengermessage); err != nil {
		fmt.Println(err)
	}
	var passagerName []string
	var pnum int = 0
	var pdetail *passengerDetail
	for _, pdetail = range getpassengermessage.Data.Datas {
		pnum++
		passagerName = append(passagerName, pdetail.Passenger_name)
	}
	return pnum, passagerName
}

func main() {
	initAll()
	fmt.Println("©wangbright\n" +
		"1、登录验证码在您的D盘根目录下，文件名为：loginPic.png，请您自己打开识别后，输入正确的序号（例如：1  或者  4,5）\n" +
		"2、铁总的API总是在变化，如果有报错，请私信\n" +
		"现在请您开始使用：\n")
	var isLogin = false
	var buyTicketName string
	//var username string    //保存登陆后返回的用户名
	for isLogin == false {
		/***********************
			输入登陆用户名密码
		    ***********************/
		fmt.Println("登录12306")
		fmt.Print("0.用户名：")
		usernameInput := ""
		_, _ = fmt.Scanln(&usernameInput)
		//fmt.Println(usernameInput)
		fmt.Print("0.密码：")
		passwardInput := ""
		_, _ = fmt.Scanln(&passwardInput)
		/*		fmt.Print("0.购票人姓名（已录铁总）：")
				buyTicketName = ""
				_, _ = fmt.Scanln(&buyTicketName)
		*/ /***********************
			验证码确认：
		    ***********************/
		var codeChk = false
		for codeChk == false {
			getCodeRe := getCode()
			if getCodeRe == false {
				continue
			}
			fmt.Print("1.验证码序列（用逗号隔开多个）：")
			strInput := ""
			_, _ = fmt.Scanln(&strInput)
			inputSplit := strings.Split(strInput, ",")
			j := 0
			code := ""
			for j < len(inputSplit)-1 {
				code += coordinates[inputSplit[j]]
				code += ","
				j++
			}
			code += coordinates[inputSplit[j]]
			codeChk = checkCode(code)
		}
		/***********************
			登陆
			   输入：用户名、密码
			   输出：成功标志，用户中文名
		    ***********************/
		isLogin = Login(usernameInput, passwardInput) //登录
		if isLogin == false {
			fmt.Println("用户名与密码验证失败，请重新输入！")

		}
	}

	request_url := "https://kyfw.12306.cn/otn/queryOrder/queryMyOrderNoComplete"
	postDict := map[string]string{
		"_json_att": "",
	}
	type stationDTODeatil struct {
		Station_train_code string `json:"station_train_code"`
		From_station_name  string `json:"from_station_name"`
		To_station_name    string `json:"to_station_name"`
	}
	type passengerDTODeatil struct {
		Passenger_name string `json:"passenger_name"`
	}

	type ticketDetails struct {
		Ticket_status_name string             `json:"ticket_status_name"`
		Sequence_no        string             `json:"sequence_no"`
		StationTrainDTO    stationDTODeatil   `json:"stationTrainDTO"`
		PassengerDTO       passengerDTODeatil `json:"passengerDTO"`
	}
	type orderdbDetails struct {
		Sequence_no     string          `json:"sequence_no"`
		Order_date      string          `json:"order_date"`
		Ticket_totalnum int             `json:"ticket_totalnum"`
		Cancel_flag     string          `json:"cancel_flag"`
		Pay_flag        string          `json:"pay_flag"`
		Tickets         []ticketDetails `json:"tickets"`
	}

	type myorderDetails struct {
		OrderDBList []orderdbDetails `json:"orderDBlist"`
	}
	type myOrderInfo struct {
		ValidateMessagesShowId string            `json:"validateMessagesShowId"`
		Status                 bool              `json:"status"`
		HttpStatus             int               `json:"httpstatus"`
		Data                   myorderDetails    `json:"data"`
		Messages               []string          `json:"messages"`
		ValidateMessages       map[string]string `json:"validateMessages"`
	}
	var myorderinfo myOrderInfo
	_, html := getUrlRespHtml(request_url, postDict)
	//fmt.Print(html)
	if err := json.Unmarshal([]byte(html), &myorderinfo); err != nil {
		fmt.Print(html)
		fmt.Println(err)
	} else {
		var orderdbdetail orderdbDetails
		//fmt.Println("hello")
		if myorderinfo.Data.OrderDBList != nil {
			fmt.Println("您的订单如下：")
			// if myorderinfo.Data!=myorderDetails.nil{
			for _, orderdbdetail = range myorderinfo.Data.OrderDBList {
				fmt.Println("订单编号 ", orderdbdetail.Sequence_no, orderdbdetail.Cancel_flag, "乘车人 ", orderdbdetail.Tickets[0].PassengerDTO.Passenger_name,
					"列车编号 ", orderdbdetail.Tickets[0].StationTrainDTO.Station_train_code, "订单状态 ", orderdbdetail.Tickets[0].Ticket_status_name)
				if orderdbdetail.Tickets[0].Ticket_status_name == "待支付" {
					fmt.Println("请前往 12306网站支付该订单，或者取消该订单！，输入Y直接在此取消，否则不处理：")
					var yesInput string
					_, _ = fmt.Scanln(&yesInput)
					if yesInput == "Y" {
						request_url := cancelNoCompleteMyOrderAPI
						postDict := map[string]string{
							"sequence_no": orderdbdetail.Sequence_no,
							"cancel_flag": "cancel_order",
						}
						_, html := getUrlRespHtml(request_url, postDict)
						fmt.Print(html)
						fmt.Println("已经取消订单")
					}

				}

			}
		} else {
			fmt.Println("您铁总系统中无订单")
		}
	}
	fmt.Println("您的铁总系统包含如下乘车人信息：")
	pnum, passslice := queryPassager()
	for i, s := range passslice {
		fmt.Print(i+1, "：", s, "   ")
	}
	var numInput int = 1
	for {
		fmt.Print("\n请选择购买的客户序号：")
		_, _ = fmt.Scanln(&numInput)
		if numInput >= 1 && numInput <= pnum {
			break
		}
		fmt.Println("请在以上序号中选择")
	}
	fmt.Println("您将为 ", passslice[numInput-1], "购买车票")
	buyTicketName = passslice[numInput-1]

	var ischkbuy = false
	var trainInfoDetails []string
	var getTrainBool bool
	dateInput := ""
	fromStationInput := ""
	toStationInput := ""
	var trainOKNm int = 0
	for ischkbuy == false {
		/***********************
		  查询火车票   输入购买日期、起始站、达到站
		  ***********************/
		fmt.Print("0.购车票日期（2018-12-22）：")
		_, _ = fmt.Scanln(&dateInput)
		for {
			fmt.Print("0.购车票起始火车站点：")
			_, _ = fmt.Scanln(&fromStationInput)
			if _, okinput := stationMap[fromStationInput]; okinput {
				break
			}
			fmt.Println("无该火车站信息，请重新输入")
		}
		for {

			fmt.Print("0.购车票达到站点：")
			_, _ = fmt.Scanln(&toStationInput)
			if _, okinput := stationMap[toStationInput]; okinput {
				break
			}
			fmt.Println("无该火车站信息，请重新输入")
		}
		/*
			查询出来火车票有3中主要状态，由第2与第11个字段确定： 第2个字段有3中状态： 1： Y ：存在某种余票。 N：没有任何票的余票了（第二个字段为预定，但是不显示，且没有余额编码）。      IS_TIME_NOT_BUY  需要读取第二字段，获取预售时间
			查询火车票
			   输入：列车日期、出发站编码、到达站编码
			   输出：列车详细信息
		*/

		getTrainBool, trainInfoDetails = queryTrain(dateInput, fromStationInput, toStationInput)
		if getTrainBool {
			fmt.Println("查询出来的火车信息如下：")
			var tickStatStr = ""
			trainOKNm = len(trainInfoDetails)
			fmt.Println("一共有：", trainOKNm, " 列火车")
			fmt.Println("序号 车次 出发 到达 历时 商务 一等 二等 高软 动卧 软卧 无座 硬卧 硬座 其他 备注")
			for t, trainDetail := range trainInfoDetails {
				res := strings.Split(trainDetail, "|")
				//如果无此类型票，则显示“--”
				for i, resInfo := range res {
					if resInfo == "" {
						res[i] = "--"
					}
				}
				if res[11] == "Y" {
					tickStatStr = "还有余票"
				} else if res[11] == "N" {
					tickStatStr = "全部售罄"

				} else if res[11] == "IS_TIME_NOT_BUY" {
					tickStatStr = res[1]
				} else {
					tickStatStr = res[11]
				}
				/*			fmt.Println(strconv.Itoa(t+1), ":   ", "可预订车次列表，", res[3], " 出发时间：", res[8], "到达时间：", res[9], "历时：", res[10],
								"  商务座：", res[32],
								"  一等座：", res[31],
								"  二等座：", res[30],
								"  高级软卧：", res[21],
								"  动卧：", res[33],
								"  软卧：", res[23],
								"  无座：", res[26],
								"  硬卧：", res[28],
								"  硬座：", res[29],
								"  其他：", res[22],
								"  备注：",tickStatStr)
							}
				*/fmt.Println(strconv.Itoa(t+1), res[3], res[8], res[9], res[10], res[32], res[31], res[30], res[21], res[33], res[23], res[26], res[28], res[29], res[22], tickStatStr)

			}

			fmt.Print("是否重新查询？（Y or N）默认为Y：")
			var queryAgain string
			_, _ = fmt.Scanln(&queryAgain)
			if queryAgain == "N" {
				ischkbuy = true
			}
		}
	}
	/***********************
	  获得用户输入，明确预定第几列火车票
	 ***********************/
	var selectOK = false
	var seatType string
	//var numInput int = 1
	for selectOK == false {
		fmt.Print("购买第几趟列车（数字）：")
		_, _ = fmt.Scanln(&numInput)
		if numInput < 1 || numInput > trainOKNm {
			selectOK = false
			continue
		}
		fmt.Print("车票类型  WZ:无座,F:动卧,M:一等座,O（字母‘欧’）:二等座,1:硬座,3:硬卧,4:软卧,6:高级软卧,9:商务座:")
		_, _ = fmt.Scanln(&seatType)
		if strings.Split(trainInfoDetails[numInput-1], "|")[seatTicketMap[seatType]] == "" {
			fmt.Println("无此类别座位，请重新选择列车与座位类别")
			selectOK = false
			continue
		}
		selectOK = true
	}
	trainName := strings.Split(trainInfoDetails[numInput-1], "|")[3]

	var buyOK = false
	for buyOK == false {
		fmt.Println("Oh，Ok，使用goroutine进行购买车票\n 购买的类型为：", dateInput, fromStationInput, toStationInput, trainName, seatTypeMap[seatType])
		ctx, cancel := context.WithCancel(context.Background())
		go spinner(ctx) //提示正在买票
		boolRe, trainDetail := routineQueryTick(dateInput, fromStationInput, toStationInput, strings.Split(trainInfoDetails[numInput-1], "|")[3], seatType)
		/***********************
			 判断用户是否在铁总系统在线：
			    输入：无
			    输出：无
		     ***********************/
		if boolRe == true {
			cancel() // 所有基于这个Context或者衍生的子Context都会收到通知
			fmt.Println("共查询了 ", searchTickTimes, "  次！")
		}
		var isChkOnline = false
		isChkOnline = chkOnline()
		if isChkOnline == true {
			fmt.Println("5：用户在铁总系统在线验证成功：")
		}
		buyOK = buyTicket(buyTicketName, trainDetail, dateInput, fromStationInput, toStationInput, seatType)
		/**
		在线验证成功：
		  输入：无
		  输出：成功则返回 true
		*/
	}

	fmt.Print("\n hello, 世界！")
	var closeInput string
	fmt.Print("现已完成，请您关闭程序")
	_, _ = fmt.Scanln(&closeInput)
}
