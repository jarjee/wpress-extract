#!/usr/bin/env bash
set -e

 # Generate directory structure
 mkdir -p testdata
 (
   cd testdata
   mkdir -p wp-content/{plugins/sample-plugin,uploads/2024,mu-plugins,themes/sample-theme,cache}
   touch {index,wp-config,object-cache,advanced-cache}.php .htaccess
   touch wp-content/plugins/sample-plugin/{sample-plugin.js,style.css,plugin.php}
   touch wp-content/uploads/testfile.txt
   touch wp-content/mu-plugins/mu-plugin.php
   touch wp-content/themes/sample-theme/{theme.js,style.css,functions.php}
 )

 # Build the wpress tool first
 go build -o wpress main.go

 # Create valid.wpress by compressing the testdata content
 ./wpress -input testdata/ -mode compress -out testdata/valid.wpress

 # Cleanup after ourselves
 rm wpress

 echo "Testdata generated successfully"
