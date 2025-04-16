package postfixutil

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	softDSN string = "5.2.0 5.2.1 5.2.2 5.3.1 5.4.5 5.5.3"
	queueIDPattern string = "\\]:\\s([A-Z0-9]+):"
	logPattern string = "^([A-Za-z]{3}\\s+\\d+ [0-9:]{8}) .*? .*?: ([A-Z0-9]+): to=<(.*?)>, relay=(.*?), delay=(.*?), delays=(.*?), dsn=(.*?), status=(.*?) "
	reasonPattern string = "said:(.*)$"
	deferredPattern string = "status=deferred \\((.*)\\)$"
	bouncedPattern string = "status=bounced \\((.*)\\)$"
	senderPattern string = "from=<(.*?)>"
)

var bounceReasons = map[string]string{
	// yahoo
	"5.0.0" : "Mailbox does not exist",

	// gmail
	"5.1.1": "Mailbox does not exist",
	"5.2.1": "Mailbox Disabled",

	"5.2.2": "Mailbox full", // icloud
	"4.2.2": "Mailbox full", // gmail,
	

	"5.4.4": "A record not found",
	// "4.4.3": "MX record not found",
	
}

type Bounce struct {
	// ID string
	Date time.Time `json:"date"`
	QueueID string `json:"queueId"`
	From string `json:"from"`
	To string `json:"to"`
	Relay string `json:"relay"`
	Delay string `json:"delay"`
	Delays string `json:"delays"`
	DSN string `json:"dsn"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type BounceLog struct {
	Date time.Time
	QueueID string
	To string
	Relay string
	Delay string
	Delays string
	DSN string
	Status string
	Reason string
}

type SenderLog struct {
	QueueID string
	From string
}



func (b *Bounce) IsHard() bool {
	return !strings.Contains(softDSN, b.DSN)
}

func FindBounces(paths *[]string) []Bounce {
	var bounces []Bounce
	bounceQueueID := make(map[string]interface{})
	var logs []string
	var rawSenderLogs []string
	var senderLogs []SenderLog
	var bounceLogs []BounceLog
	queueIDRegex, _ := regexp.Compile(queueIDPattern)
	logRegex, _ := regexp.Compile(logPattern)
	senderRegex, _ := regexp.Compile(senderPattern)
	reasonRegex, _ := regexp.Compile(reasonPattern)
	deferredRegex, _ := regexp.Compile(deferredPattern)
	bouncedRegex, _ := regexp.Compile(bouncedPattern)

	for _, path := range *paths {
		file, err := os.Open(path)
		if err != nil {
			log.Fatalln(err)
			return bounces
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			log := scanner.Text()
			// if strings.Contains(log, "postfix/bounce") {
			if strings.Contains(log, "postfix/qmgr") {
				queue := queueIDRegex.FindStringSubmatch(log)
				if len(queue) >= 2 {
					queueID := queue[1]
					bounceQueueID[queueID] = nil
				}
				
				if len(senderRegex.FindStringSubmatch(log)) == 2 {
					rawSenderLogs = append(rawSenderLogs, log)
				}
			} else if strings.Contains(log, "status=deferred") || strings.Contains(log, "status=bounced") {
				logs = append(logs, log)
			} 
		}
	}

	for _, l := range logs {
		queueID := queueIDRegex.FindStringSubmatch(l)[1]
		if _, exists := bounceQueueID[queueID]; exists {
			v := logRegex.FindStringSubmatch(l)
			
			// Extract full bounce response
			reason := ""
			
			// Try different patterns to extract the reason
			if reasonMatch := reasonRegex.FindStringSubmatch(l); len(reasonMatch) > 1 {
				reason = strings.TrimSpace(reasonMatch[1])
			} else if deferredMatch := deferredRegex.FindStringSubmatch(l); len(deferredMatch) > 1 {
				reason = strings.TrimSpace(deferredMatch[1])
			} else if bouncedMatch := bouncedRegex.FindStringSubmatch(l); len(bouncedMatch) > 1 {
				reason = strings.TrimSpace(bouncedMatch[1])
			} else if _, exists := bounceReasons[v[7]]; exists {
				reason = bounceReasons[v[7]]
			}
			
			wib, err := time.LoadLocation("Asia/Jakarta")
			if err != nil {
				log.Fatalln("Error loading WIB location:", err)
			}
		
			if len(v) >= 9 {
				// fmt.Println(v[2])
				bounceLogs = append(bounceLogs, BounceLog{ParseDate(v[1],wib), v[2], v[3], v[4], v[5], v[6], v[7], v[8], reason})
			} 
		}
	}


	for _, l := range rawSenderLogs {
		queueID := queueIDRegex.FindStringSubmatch(l)[1]
		if _, exists := bounceQueueID[queueID]; exists {
			v := senderRegex.FindStringSubmatch(l)
			
			if len(v) == 2 {
				// if duplicate, remove
				exists := false
				for _, senderLog := range senderLogs {
					if senderLog.QueueID == queueID {
						exists = true
						break
					}
				}
				if !exists {
					senderLogs = append(senderLogs, SenderLog{queueID, v[1]})
				}


			} else {
				fmt.Printf("Error parsing sender: %v\n", l)
			}
		}
	}

	for _, bounceLog := range bounceLogs {
		for _, senderLog := range senderLogs {
			if bounceLog.QueueID == senderLog.QueueID {
				bounces = append(bounces, Bounce{bounceLog.Date, bounceLog.QueueID,senderLog.From, bounceLog.To, bounceLog.Relay, bounceLog.Delay, bounceLog.Delays, bounceLog.DSN, bounceLog.Status, bounceLog.Reason})
			}
		}
	}

	return bounces
}

func ParseDate(Date string, location *time.Location) time.Time {
	value := fmt.Sprintf("%v %v", time.Now().Year(), Date)

	localTime, err := time.ParseInLocation("2006 Jan 2 15:04:05", value,location)
	if err != nil {
		log.Fatalln("Error Parsing Time Format: ", value)
		return time.Now()
	}

	utcTime := localTime.UTC()
	return utcTime
}

func DeleteQueue(email string) error {
	command := fmt.Sprintf("/opt/zimbra/common/sbin/postqueue -p | tail -n +2 | awk 'BEGIN { RS = \"\" } /%s/ { print $1 }' | tr -d '*!' | /opt/zimbra/common/sbin/postsuper -d -", email)

	cmd := exec.Command("bash", "-c", command)

	_, err := cmd.CombinedOutput()
	return err
}
