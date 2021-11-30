# HNT-HMRC

## Calcuate your UK income tax liability from Helium mining

You can use this code by visiting the [site](https://hnt-hmrc.herokuapp.com)


### FAQ
#### How does it work
This project does the following

- Fetch all of the hotspots linked to a wallet
- Compile their daily earnings in GBP (based on the HNT value on the day via Coingecko)
- Provides a summary, as well as a CSV of the income

#### Why did you do this?
I wrote this as week script for myself, and thought the community might find it useful

#### How does it make money?
It doesn't, thankfully it's hosted on heroku's free tier so it doesn't cost me anything to run. Donations are appreciated.

#### Can I contribute?
Code, sure! Create a PR and I'll take a look. This isn't my day job, so don't expect the best SLA

### Running Locally

```
go run .
````

