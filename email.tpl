<h2>Welcome to Secret Santa 2019!</h2>

<div>
    <p>You are the secret santa for <b>{{.Name}}</b>!</p>
    <img style="float: left; margin-right: 10px" src="https://github.com/sndurkin/secret-santa/raw/master/secret-santa.png" />
    <div style="float: left; max-width: 400px">
      <p>Here is {{.Pronoun}} wishlist:</p>
      <ul>
        {{range .Wishlist}}
        <li>{{.}}</li>
        {{end}}
      </ul>
    </div>
</div>
