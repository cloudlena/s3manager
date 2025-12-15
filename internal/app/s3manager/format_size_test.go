package s3manager

import "testing"

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 bytes"},
		{1, "1 bytes"},
		{512, "512 bytes"},
		{1023, "1023 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
		{1099511627776, "1.00 TB"},
		{1649267441664, "1.50 TB"},
	}

	for _, test := range tests {
		result := FormatFileSize(test.bytes)
		if result != test.expected {
			t.Errorf("FormatFileSize(%d) = %q; want %q", test.bytes, result, test.expected)
		}
	}
}
