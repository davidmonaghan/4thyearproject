#! /bin/bash

if [ $# -ne 4 ]
then
        echo "Too few arguments provided"
        exit 1
fi

#uWebSiteName=$1 # URL for website
newDBrootPW=$1  # New root password for MySQL
uDBsitename=$2  # Wordpress installation database name
uDBuname=$3     # MySQL username for wordpress installation database
uDBpassword=$4  # MySQL password for wordpress installation database

# ***********************************************************************************
# 1. Change root password an MySQL database.
# 2. Create user/password/database for website CMS. 
# ***********************************************************************************

oldDBrootPW="root"
db="SET PASSWORD FOR root@localhost = PASSWORD('$newDBrootPW');create database $uDBsitename;GRANT ALL PRIVILEGES ON $uDBsitename.* TO $uDBuname@localhost IDENTIFIED BY '$uDBpassword';FLUSH PRIVILEGES;"
mysql -u root -p$oldDBrootPW -e "$db"

# ***********************************************************************************
# 3. Update Wordpress wp-config file.
# 4. Give ownership of /var/www/site folder Apache2.
# ***********************************************************************************

#sed -i -e 's/database_name_here/'${uDBsitename}'/g' /var/www/html/wp-config.php
#sed -i -e 's/username_here/'${uDBuname}'/g' /var/www/html/wp-config.php
#sed -i -e 's/password_here/'${uDBpassword}'/g' /var/www/html/wp-config.php

exit 0
