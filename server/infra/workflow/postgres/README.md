This is code from cschleiden/go-workflowsor more precisely from pull request #458, modified a bit for needs of this project.
I used LLM here to make it. I hope to replace it in future with proper implementation when LISTEN NOTIFY is implemented in
cschleiden/go-workflows. Until then, this will stay.

Why?
- Postgres cpu usage went from ~1-2% to basically 0% when nothing is happening
- Delay between activities went from 300ms to 15ms

Entire point of this app is to have lightweight alternative to sonarr/radarr, so this is important.