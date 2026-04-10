# xfile

정적 파일 호스팅 서비스에 파일을 업로드하기 위한 간단한 CLI 도구입니다.

## 사용법

### 기본 업로드

파일을 직접 업로드:

```bash
xfile /path/to/your/file.txt
```

또는 `--file` 플래그 사용:

```bash
xfile --file /path/to/your/file.txt
```

## 명령줄 옵션

- `<file-path>` (위치 인자): 업로드할 파일의 경로
- `--file`: 업로드할 파일의 경로 (위치 인자 대신 사용 가능)
- `--verbose`: 상세 출력 활성화
- `--version`: 버전 정보 표시

## 예제

```bash
# 이미지 업로드 (위치 인자)
xfile photo.jpg

# 상세 출력과 함께 업로드
xfile document.pdf --verbose

# 버전 확인
xfile --version
```

## API

이 도구는 https://static.a85labs.net/docs 에 있는 API와 함께 작동하도록 설계되었습니다.

이 도구는 API가 다음을 수행할 것으로 예상합니다:
- `/upload` 엔드포인트로 POST 요청 수락
- multipart/form-data 파일 업로드 지원
- 업로드된 파일 URL이 포함된 JSON 응답 반환

## 라이선스

AGPL-3.0
