# Postfix Bounce Parser

A Go tool for analyzing Postfix email server bounce logs and generating reports
in JSON and Excel formats.

## Overview

This utility scans Postfix mail server logs to identify email bounces, extracts
relevant information, and outputs structured data in both JSON and Excel
formats. It's useful for email administrators who need to analyze and report on
email delivery failures.

## Features

- Scans Postfix log files for bounce information
- Extracts details such as sender, recipient, bounce reason, and DSN codes
- Classifies bounces (hard vs soft)
- Outputs data in JSON format for programmatic use
- Generates Excel spreadsheets for easy analysis and reporting
- Supports scanning directories recursively

## Requirements

- Go 1.21+
- Postfix mail server logs
- Environment variable setup in `.env` file

## Installation

```bash
git clone https://github.com/agldw/postfix-bounce-parser.git
cd postfix-bounce-parser
go mod tidy
```

## Configuration

Create a `.env` file in the project root with the following:

```
LOG_DIR=/path/to/your/postfix/logs
```

## Usage

Simply run the binary:

```bash
go run main.go
```

The tool will:

1. Scan all files in the directory specified by `LOG_DIR` (including
   subdirectories)
2. For each file containing bounce information:
   - Create a JSON file with the same name plus `.json` extension
   - Create an Excel file with the same name plus `.xlsx` extension

## Output Format

### JSON

Each bounce record contains:

- `date`: Timestamp of the bounce event
- `queueId`: Postfix queue ID
- `from`: Sender email address
- `to`: Recipient email address
- `relay`: Server that reported the bounce
- `delay`: Delivery delay time
- `delays`: Breakdown of delay components
- `dsn`: Delivery Status Notification code
- `status`: Delivery status
- `reason`: Human-readable bounce reason

### Excel

Excel files contain the same data in spreadsheet format with the following
columns:

- Date
- From
- To
- Relay
- Delay
- DSN
- Status
- Reason

## License

MIT

## Contributing

Contributions welcome! Please feel free to submit a Pull Request.
