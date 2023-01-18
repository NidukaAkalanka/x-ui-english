پنل فارسی X-UI

به زودی توضیحات فارسی تکمیل میشود.


یکی دیگر از نسخه های ترجمه شده به انگلیسی و فارسی X-UI. با برخی از ویژگی های پیشرفته تر پیاده سازی شده است.


# امکانات

- فارسی و انگلیسی شده در همه قسمت ها (تنظیمات سمت سرور + رابط کاربری سمت سرور + رابط کاربری وب)
- نظارت بر وضعیت سیستم
- پشتیبانی از پروتکل چند کاربره، web page visualization operation
- چند UUID را می توان به عنوان کاربر برای تنظیمات Vmess و Vless با کدهای QR مجزا اضافه کرد.
- محدود کردن IP
- پشتیبانی از پروتکل های: vmess, vless, trojan, shadowsocks, dokodemo-door, socks, http
- پشتیبانی برای پیکربندی تنظیمات انتقال بیشتر
- آمار ترافیک, محدود کردن ترافیک, محدودیت با زمان انقضا 
- قالب های پیکربندی xray قابل تنظیم
- پشتیبانی از پنل دسترسی https (نام دامنه + گواهی ssl خود را می توان استفاده کرد)
- ربات تلگرام برای توابع اولیه و اطلاعیه ها
- پشتیبانی از برنامه گواهینامه SSL با یک کلیک و تمدید خودکار
- می توان به طور ایمن از v2-ui مهاجرت کرد 
- می تواند به طور ایمن از نسخه قبلی X-UI (CH/EN) بدون از دست دادن خروجی به روزرسانی شود
- برای موارد پیکربندی پیشرفته تر، برای جزئیات بیشتر به پنل مراجعه کنید

# پیش نمایش
![](media/Web.png)
![](media/PostInstallation.png)
# پیش نمایش بات تلگرام (Currently, only for V0.2)
![](media/TGBot1.PNG)![](media/TGBot2.PNG)

# نصب و ارتقاء تک دستوره

```
bash <(curl -Ls https://raw.githubusercontent.com/NidukaAkalanka/x-ui-english/master/install.sh)
````
## نصب و ارتقاء دستی

1. ابتدا سیستم خود را آپدیت کنید و دستورات زیر را اجرا کنید. (باید مجوزهای کاربر روت را داشته باشد)
```` 
sudo su
cd
````
2. سپس آخرین بسته فشرده را از https://github.com/NidukaAkalanka/x-ui-english/releases/latest دانلود کنید، به طور معمول معماری «amd64» را انتخاب کنید.

3. دستورات زیر را به ترتیب اجرا کنید:

> اگر معماری cpu سرور شما "amd64" نیست، "*" را در دستور با معماری دیگری جایگزین کنید درعیر اینصورت همان "amd64" را وارد کنید. 
````
rm x-ui/ /usr/local/x-ui/ /usr/bin/x-ui -rf
tar zxvf x-ui-linux-amd64.tar.gz
chmod +x x-ui/x-ui x-ui/bin/xray-linux-* x-ui/x-ui.sh
cp x-ui/x-ui.sh /usr/bin/x-ui
cp -f x-ui/x-ui.service /etc/systemd/system/
mv x-ui/ /usr/local/
systemctl daemon-reload
systemctl enable x-ui
systemctl restart x-ui
````

## نصب با استفاده از docker
1. ابتدا docker را نصب کنید
```shell
curl -fsSL https://get.docker.com | sh
````
2. سپس x-ui را نصب کنید
```shell
mkdir x-ui && cd x-ui
docker run -itd --network=host \
    -v $PWD/db/:/etc/x-ui/ \
    -v $PWD/cert/:/root/cert/ \
    --name x-ui --restart=unless-stopped \
    enwaiax/x-ui:latest
````

> ایمیج خود را بسازید
 ```shell
docker build -t x-ui .
````

