# Icon font

`material-symbols-outlined.woff2` is a **subset** of Google's Material Symbols
Outlined containing only the icons this app renders, plus the handful BeerCSS
needs internally (checkbox, radio and switch glyphs). BeerCSS's `@font-face`
loads it by that exact filename, relative to `beer.min.css` — so it has to stay
next to it, under that name.

If you add an `<i>new_icon</i>` to a template, regenerate the subset with the
new name appended to the list:

```sh
NAMES="add,archive,arrow_back,arrow_downward,arrow_drop_down,arrow_upward,check,check_box,check_box_outline_blank,check_circle,chevron_left,chevron_right,close,content_copy,create_new_folder,dark_mode,database,delete,description,download,drive_folder_upload,edit,error,expand_less,expand_more,first_page,folder,folder_open,indeterminate_check_box,info,insert_drive_file,last_page,light_mode,link,more_vert,music_note,open_in_new,photo,public,radio_button_checked,radio_button_unchecked,search,sort,upload,warning"

curl -sA Mozilla/5.0 \
  "https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@24,400,0,0&icon_names=$NAMES&display=block" \
  | grep -o 'https://fonts.gstatic.com[^)]*' \
  | xargs curl -s -o web/static/css/material-symbols-outlined.woff2
```

An icon that is missing from the subset renders as its literal name.
