package main

import (
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestCappedBuffer(t *testing.T) {
	Scenario(t, "GIVEN 상한보다 적게 기록함 WHEN 읽음 THEN 전부 그대로 나온다", func(t *testing.T) {
		b := newCappedBuffer(100)
		_, _ = b.Write([]byte("hello"))
		if got := b.String(); got != "hello" {
			t.Errorf("got %q, 기대값 %q", got, "hello")
		}
	})

	// 경계 케이스: 딱 상한만큼 — 아직 아무것도 버리지 않았으니 생략 표시가 붙으면 안 된다.
	Scenario(t, "GIVEN 상한을 정확히 채움 WHEN 읽음 THEN 생략 표시 없이 전부 나온다 (경계 케이스)", func(t *testing.T) {
		b := newCappedBuffer(100)
		full := strings.Repeat("a", 100)
		_, _ = b.Write([]byte(full))
		if got := b.String(); got != full {
			t.Errorf("상한 정확히 채운 경우 그대로여야 하는데 %q", got)
		}
	})

	Scenario(t, "GIVEN 상한보다 훨씬 많이 기록함 WHEN 읽음 THEN 최근 부분만 남고 앞에 생략 표시가 붙는다", func(t *testing.T) {
		b := newCappedBuffer(100)
		_, _ = b.Write([]byte("START" + strings.Repeat("x", 1000) + "END"))
		got := b.String()
		if !strings.HasPrefix(got, cappedOmittedNotice) {
			t.Errorf("생략 표시가 앞에 있어야 하는데 %q", got[:min(len(got), 30)])
		}
		body := strings.TrimPrefix(got, cappedOmittedNotice)
		if len(body) > 100 {
			t.Errorf("본문이 상한(100)을 넘음: %d", len(body))
		}
		if !strings.HasSuffix(body, "END") || strings.Contains(body, "START") {
			t.Errorf("최근 내용(END)은 남고 오래된 내용(START)은 버려져야 하는데 %q", body)
		}
	})

	// 회귀 테스트: 마운트가 며칠씩 살아 있는 동안 rclone이 계속 로그를 쓰면
	// 이전 구현(bytes.Buffer)은 끝없이 커졌다.
	Scenario(t, "GIVEN 매우 많은 쓰기가 누적됨 WHEN 내부 메모리 확인 THEN 상한의 몇 배를 넘지 않는다 (무한 증가 회귀 테스트)", func(t *testing.T) {
		b := newCappedBuffer(1000)
		chunk := []byte(strings.Repeat("e", 100))
		for i := 0; i < 20000; i++ { // 약 2MB 쓰기
			_, _ = b.Write(chunk)
		}
		if len(b.buf) > 3*1000 {
			t.Errorf("내부 버퍼가 상한(1000)의 3배를 넘음: %d", len(b.buf))
		}
	})

	// exec.Cmd는 stderr 쓰기가 짧게 끝나면 오류로 취급한다 — 항상 전체 길이를 돌려줘야 한다.
	Scenario(t, "GIVEN 상한을 넘는 쓰기 WHEN Write THEN 입력 길이를 그대로 반환하고 오류가 없다", func(t *testing.T) {
		b := newCappedBuffer(10)
		n, err := b.Write([]byte(strings.Repeat("z", 500)))
		if n != 500 || err != nil {
			t.Errorf("got (%d, %v), 기대값 (500, nil)", n, err)
		}
	})

	// 부정 케이스: 한글(3바이트)을 바이트 단위 상한에서 자르면 글자 중간에서 끊길 수 있다.
	Scenario(t, "GIVEN 한글이 가득한 출력을 상한에서 자름 WHEN 읽음 THEN 깨진 UTF-8이 없다 (부정 케이스)", func(t *testing.T) {
		for _, max := range []int{7, 10, 100, 101, 102} {
			b := newCappedBuffer(max)
			_, _ = b.Write([]byte(strings.Repeat("가나다", 200)))
			if got := b.String(); !utf8.ValidString(got) {
				t.Errorf("max=%d 에서 깨진 UTF-8: %q", max, got)
			}
		}
	})

	Scenario(t, "GIVEN 상한이 0 이하로 주어짐 WHEN 생성해서 사용 THEN 최소값으로 보정되어 panic 없이 동작한다 (경계 케이스)", func(t *testing.T) {
		for _, max := range []int{0, -5} {
			b := newCappedBuffer(max)
			_, _ = b.Write([]byte("abc"))
			if got := b.String(); !strings.HasSuffix(got, "c") {
				t.Errorf("max=%d: 최근 내용은 남아야 하는데 %q", max, got)
			}
		}
	})

	Scenario(t, "GIVEN 여러 고루틴이 동시에 기록하고 읽음 WHEN -race THEN 데이터 레이스가 없다", func(t *testing.T) {
		b := newCappedBuffer(256)
		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 500; i++ {
					_, _ = b.Write([]byte("line of rclone output\n"))
					_ = b.String()
				}
			}()
		}
		wg.Wait()
	})
}
