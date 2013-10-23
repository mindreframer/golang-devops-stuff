Creating an app
---------------

    shipbuilder apps:create MyApp python

    shipbuilder config:set DATABASE_URL='postgres://example.com:5432/mydb' TWILIO_APP_SID='SOME_APP_SID' -aMyApp

    shipbuilder domains:add sb-staging.sendhub.com -aMyApp

    cd path/to/MyApp

    git remote add sb ssh://ubuntu@YOUR-SB-SERVER/git/MyApp

    git push -f sb
