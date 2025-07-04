// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package crowdstrike

import (
	"github.com/elastic/beats/v7/libbeat/asset"
)

func init() {
	if err := asset.SetFields("filebeat", "crowdstrike", asset.ModuleFieldsPri, AssetCrowdstrike); err != nil {
		panic(err)
	}
}

// AssetCrowdstrike returns asset data.
// This is the base64 encoded zlib format compressed contents of module/crowdstrike.
func AssetCrowdstrike() string {
	return "eJy8m19v47gRwN/3Uwxwr72gXWCvRR4KGHayazTOBrG317eCocYWG4qjIyn7/O0LkpIi27Il2dT6KdCf4U9Dcv4yv8I77u+Ba9olxmrxjp8ArLAS72HqLi6rixolMoP38IaWfQJI0HAtcitI3cM/PwEALCgpJMKaNHCSErkVahPEBNmAW1TW3H0CWAuUibn37/0KimV4TOF+dp/jPWw0FXl5pWVY93v04vzQzfEemeSkwrDAVAJMoraQMMvuynebIE2YDC1zz9U3ar0syjvlq40HzsB53aBl0HjLoyLjaQlnU2ZBKC6LBP1ne1wrMjSWZfldE+NEKW3f0fwWP8Rqn+PB3UrSO+53pJOjexe+xf1maN38kloWWcb0/sEN8Rd4FBp3TMoFszwtr80VFwkqe/jkK2Zk8RVNTsrgEo1xwizT9tIDDyopb08Km064FVth95MiEdVrpOGHQX16q/6E8yqaamTum1Yia1dVwuzxjQ49rVL0swg2Faaca+K80BoTIAU2RUCV5CSUm3/4sZrCj+f5f/67WLolkjF7dxmc1muDtpVWKIsb1MOAv3t5oIrsDXVYllYz/m48qiTuNQS0Duj+g4QCYzWy7A5W7jOFgcJgApbAz7xY76FQ4o8CIanWTcMWXPg6XhhLGer5zBkitYm3gKel5IpQNBTVirJF7VZgPIIlTzFjJ3JPLBEerN7aDPlFPcgGhTcuGKHaQt5mbl40cTTGb+bIWykPosE42WFjXbNpSsIHlYzEZ1FnQoWtcivl/Hhd3bC7S5EwnzmHzmzYpG4n1xuzC4ppVHYMNC+41uD1hFPK8sKifmZnpvaq3eqkVVaPlyPALkWNh3C1de+gdE4qLqGT6IUDM4a48LrbCZsO0l5w7OPpbhjH7ENwzNilvj2Uaolb1MLu4y37SiIYTnqwlqq3487XEZPFP7tM1aOQGJfBSQyruVRJY1FXNsK5ryHacjJfmE3jUTppdSz0J/LCsjd52wacUpYxlTwJFVGbDx9suUP2VDyMBFIoBKY3RdYjIlt+m/wtdizmZIIpssiaXH6bfP7y2wiwn7/8NgLuYvYlNuti9mUMUMZToXBGGRMxzbKXV2/qLIxyE2nI/7+RsU9CvUd0ta9PLirZCtwdeX+hykE7DbcypM/ET9cxhQRrPmvVWFkJMX7YXq7357H1nc0n4kzOX+JhzV+AJYl2rqTcISkZe9vemEwnQWTEXTyZRudcMW4Fj8g4X70+gPVSgTOLG9L7odHMCnnqV0p0rkrw1Wjf3/7nntvGREObUuJA+ude1qJWM2FyMsK9MEp8PAmmzLJ3VPC272fQTtn+zeSZmbwqUD5vQ1iDdzDlo2Sb9r1KfsoHRq1OGgiVCM589TuwmV5wS3tac7hhGn9P0aYYnKkoa7AuGsiY3oMwQDkqX/IhtSHHShq4JNOZutYF3cgFnuVHUacKAmruRv2kX/Gkgoxb43koi/IRAB+FwqXLr1rR1pLYwLXnhXmwCqpHCSKml3/ImJAfnkpDYVCfc/q+4tgHMB/F37chOdwuN5Cj9lW9uMluqMya4s3J6Iwc9VbwyNl2KbRFLVXXoouq4Pxc0PNGJJEdO6iuuQo2lLRzkLvSlJEGRbbZS9kxAyaMvS5k13paTVdVRy2SQajlXVDdcNvgu1b/wr13oe1KVWgsDq6p+Kq/b+rsUCPwlKkNJg6w90x/lBLM71pYi+2RxzV8IfHApJGoGtiFQVy+lQjz7gISVlWAOreKbxrGtHGlSGfVyghS+z4l6LJRCSY80YHmclL3l5+SeHhO7EHVrMqiK39V0vaDrB18zB3T4uUPoYZvl9LHx6Q8cfS3Mj4xi5rJBW0xO+znfWBKOin+dGCWUiErxYa+3JAgILRd5hnbYPwiri+Plu2b/KC/04tqlFLoUZ/pqiLoV81Ukv8szW0+RuupvgbfKDr8ekp0nSLn36dxz6U0Di2FYzRhKzRiCU5ZrikTpivQmn+fnk9hb4bbOtFX0P3ScsamFb864BDT+7UcmuisJbq4MiZDkAikYJcK3kgm+nZe5/k24j7VZImT9DOp0O5Iv4PGPwo0XZZ3SkqFss9M6PBHRC1VIg/AeD1kl0eNf2KsufhzTVuRuEAvHH7rznpcYBPXwp6ESpK486QhYOpaQ9PFy5SSiDivj9PPf//HX71kcKKDK+/BEXeaDjhW3oL24RjHC47RBH6ZzyIFX5NTGJEM5vFdjUs9A3Ec0w7qagxZ1R7lhXSs6NSJGsoQjn2Op48yjh8CM4JGhlEUEieR/UMVRIAuJJZV6h4Yo/QbDlkaD/cAemSZkPszWzoCzdrLB9FlAB3LV01FHtsANmH8IUw/YA+aMUF6IsQM+w4BOufDB8dTKqJl2s/hQDStYX1AkrmBsCvP+cBZCsXxiRn7ink8u9JBB8aNGqIcZixoP3ZXNzZySeVwBq0Wmw1qbP0nh6OM55ksuHxDo9yDKTQCe6PClm62HrTujq9JStoJtTk9Ht3ov0i2MXe+0tr6gVdVrZtBLnOiYS3ZpqsT5EmeqP3gz80ckjb9KRakhKXj1mwkkiwI70NT5VUj5x95OUwHzXPIn140rYUcKSeqcrQ8DNKlH5KCRz72eaAhL7+PoQ8kMT1wC0envV9aZouIZ26aDMbL7rKWGnEsJViNF1zeL5f/VaqVtqwKxlRYKRF04avj7KTV8P8AAAD//23ikaA="
}
