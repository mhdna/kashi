package api

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// likeMatches is Postgres ILIKE for the two wildcards the patterns use.
func likeMatches(pattern, text string) bool {
	var re strings.Builder
	re.WriteString("(?is)^")
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; r {
		case '\\':
			i++
			re.WriteString(regexp.QuoteMeta(string(runes[i])))
		case '%':
			re.WriteString(".*")
		case '_':
			re.WriteString(".")
		default:
			re.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	re.WriteString("$")
	return regexp.MustCompile(re.String()).MatchString(text)
}

func TestWildcardPattern(t *testing.T) {
	require.Equal(t, "", wildcardPattern("  "))
	require.Equal(t, "%Maya%", wildcardPattern(" Maya "))
	require.Equal(t, "Maya%", wildcardPattern("Maya*"))
	require.Equal(t, "%sen", wildcardPattern("*sen"))
	require.Equal(t, "Ma%a", wildcardPattern("Ma*a"))
	require.Equal(t, `%50\%\_off%`, wildcardPattern("50%_off"), "the term's own % and _ stay literal")
	require.Equal(t, "%", wildcardPattern("*"))
	require.Equal(t, "%Maya%", likePattern("*Maya*", "exact"), "the product box turns * into a wildcard too")
	require.Equal(t, "SH%12%", likePattern("SH*12", "starts"))
}

func TestPhonePattern(t *testing.T) {
	require.Equal(t, "", phonePattern(""))
	require.Equal(t, "", phonePattern("Maya"))
	require.Equal(t, "", phonePattern("Maya 3*"), "a letter anywhere leaves only the name")
	require.Equal(t, "%70123456%", phonePattern(" +961 70-123 456 "[6:]))
	require.Equal(t, "%961701%", phonePattern("+961 701"))
	require.Equal(t, "961%", phonePattern("961*"))
	require.Equal(t, "%456", phonePattern("*456"))
	require.Equal(t, "%961%456%", phonePattern("*961*456*"))
	require.Equal(t, "961%456", phonePattern("961*456"))
	require.Equal(t, "", phonePattern("*"), "no digits, no phone match")
}

// Randomized: a typed term finds exactly the names a person means by it.
func TestWildcardPatternRandomized(t *testing.T) {
	alphabet := []rune("abAB1%_\\")
	word := func(min, max int) string {
		out := make([]rune, min+rand.Intn(max-min+1))
		for i := range out {
			out[i] = alphabet[rand.Intn(len(alphabet))]
		}
		return string(out)
	}
	for run := 0; run < 1000; run++ {
		name := []rune(word(2, 10)) // no z in the alphabet, so "zz"+name never starts like name
		i := rand.Intn(len(name))
		j := i + 1 + rand.Intn(len(name)-i)
		piece := string(name[i:j])

		require.True(t, likeMatches(wildcardPattern(piece), string(name)), "contains %q in %q", piece, string(name))
		require.True(t, likeMatches(wildcardPattern(string(name[:j])+"*"), string(name)), "starts %q", string(name))
		require.True(t, likeMatches(wildcardPattern("*"+string(name[i:])), string(name)), "ends %q", string(name))
		require.True(t, likeMatches(wildcardPattern(string(name[:i])+"*"+string(name[j:])), string(name)), "hole %q", string(name))
		// With a * the term is anchored: extra letters in front of a "starts" term do not match.
		require.False(t, likeMatches(wildcardPattern(string(name[:j])+"*"), "zz"+string(name)), "anchored start %q", string(name))
		require.False(t, likeMatches(wildcardPattern("*"+string(name[i:])), string(name)+"zz"), "anchored end %q", string(name))
		// Without one it is a plain contains, whatever the term holds.
		other := word(1, 3)
		require.Equal(t, strings.Contains(strings.ToLower(string(name)), strings.ToLower(other)), likeMatches(wildcardPattern(other), string(name)), "term %q in %q", other, string(name))
	}
}

func getAuthed(t *testing.T, server *Server, url string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(recorder, request)
	return recorder
}

// The salesperson and client boxes reach the page and its count as patterns.
func TestListBranchInvoicesSalespersonAndWildcardClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		SalespersonPattern: "Rana%", ClientPattern: "Ma%ya", ClientDigits: "", PageSize: 14,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{
		SalespersonPattern: "Rana%", ClientPattern: "Ma%ya",
	}).Return(int64(1), nil)
	recorder := getAuthed(t, newTestServer(t, store), "/branch_invoices?page_size=14&salesperson=Rana*&client=Ma*ya")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"branch_invoices":[],"total":1}`, recorder.Body.String())
}

func TestSuggestBranchInvoiceFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().SuggestSalespersonNames(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.SuggestSalespersonNamesParams) ([]string, error) {
			require.Equal(t, "Ra%", arg.Pattern)
			require.Equal(t, int64(7), arg.BranchID.Int64)
			require.True(t, arg.BranchID.Valid)
			require.EqualValues(t, suggestionLimit, arg.MaxRows)
			return []string{"Rana", "Rami"}, nil
		})
	recorder := getAuthed(t, server, "/branch_invoices/suggest?field=salesperson&q=Ra*&branch_id=7")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"suggestions":[{"value":"Rana","hint":""},{"value":"Rami","hint":""}]}`, recorder.Body.String())

	store.EXPECT().SuggestInvoiceClients(gomock.Any(), db.SuggestInvoiceClientsParams{
		ClientPattern: "%70123%", ClientDigits: "%70123%", MaxRows: suggestionLimit,
	}).Return([]db.SuggestInvoiceClientsRow{{Name: "Maya", Phone: "+96170123456"}}, nil)
	recorder = getAuthed(t, server, "/branch_invoices/suggest?field=client&q=70123")
	require.JSONEq(t, `{"suggestions":[{"value":"Maya","hint":"+96170123456"}]}`, recorder.Body.String())

	store.EXPECT().SuggestInvoiceProducts(gomock.Any(), db.SuggestInvoiceProductsParams{
		ProductField: "name", Pattern: "%shirt%", MaxRows: suggestionLimit,
	}).Return([]db.SuggestInvoiceProductsRow{{Code: "SH1", Name: "Linen Shirt"}}, nil)
	recorder = getAuthed(t, server, "/branch_invoices/suggest?field=product&product_field=name&q=shirt")
	require.JSONEq(t, `{"suggestions":[{"value":"Linen Shirt","hint":"SH1"}]}`, recorder.Body.String())

	store.EXPECT().SuggestInvoiceProducts(gomock.Any(), gomock.Any()).Return([]db.SuggestInvoiceProductsRow{{Code: "SH1", Name: "Linen Shirt"}}, nil)
	recorder = getAuthed(t, server, "/branch_invoices/suggest?field=product&q=SH")
	require.JSONEq(t, `{"suggestions":[{"value":"SH1","hint":"Linen Shirt"}]}`, recorder.Body.String(), "the code is the value unless searching names")

	// Nothing typed, or a barcode field, asks the database nothing.
	for _, url := range []string{"/branch_invoices/suggest?field=client&q=", "/branch_invoices/suggest?field=client&q=%20%20", "/branch_invoices/suggest?field=product&product_field=barcode&q=529"} {
		recorder = getAuthed(t, server, url)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.JSONEq(t, `{"suggestions":[]}`, recorder.Body.String(), url)
	}
	require.Equal(t, http.StatusBadRequest, getAuthed(t, server, "/branch_invoices/suggest?q=a").Code, "a field is required")
	require.Equal(t, http.StatusBadRequest, getAuthed(t, server, "/branch_invoices/suggest?field=nope&q=a").Code)
}
