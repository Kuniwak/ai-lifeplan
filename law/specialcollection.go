package law

import "github.com/Kuniwak/lifeplan/money"

const SpecialCollectionPensionFloor money.Yen = 180_000

type SpecialCollectionSubject struct {
	Pension money.Yen

	NursingCare, LateElderly money.Yen

	NursingCareLastRegular, LateElderlyLastRegular money.Yen

	JoinedThisYear bool
}

type SpecialCollection struct {
	NursingCare, LateElderly bool
}

func SpeciallyCollectedFrom(s SpecialCollectionSubject) SpecialCollection {
	if s.JoinedThisYear || s.Pension < SpecialCollectionPensionFloor {
		return SpecialCollection{}
	}

	nursingCare := s.NursingCare > 0
	if !nursingCare {
		return SpecialCollection{}
	}

	if s.LateElderly <= 0 {
		return SpecialCollection{NursingCare: true}
	}

	careOctober, _ := RegularInstalmentOf(s.NursingCare, s.NursingCareLastRegular)
	koukiOctober, _ := RegularInstalmentOf(s.LateElderly, s.LateElderlyLastRegular)
	october := careOctober + koukiOctober

	return SpecialCollection{
		NursingCare: true,
		LateElderly: october <= HalfOfAPensionPayment(s.Pension),
	}
}

const (
	PensionPaymentsAYear = 6
	ProvisionalPayments  = 3
	RegularPayments      = 3
	InstalmentUnit       = money.Yen(100)
)

func ProvisionalInstalmentOf(yearly money.Yen) money.Yen {
	return (yearly / 12 * 6 / ProvisionalPayments).Truncate(InstalmentUnit)
}

func RegularInstalmentOf(yearly, provisionalEach money.Yen) (october, later money.Yen) {
	rest := max(yearly-provisionalEach*ProvisionalPayments, 0)

	later = (rest / RegularPayments).Truncate(InstalmentUnit)
	return rest - later*(RegularPayments-1), later
}

func HalfOfAPensionPayment(pension money.Yen) money.Yen {
	return (pension / PensionPaymentsAYear) / 2
}

type LastRegularInstalments struct {
	NursingCare, LateElderly money.Yen

	NursingCareStarted, LateElderlyStarted bool
}

func (s SpecialCollectionSubject) Next(withheld SpecialCollection) LastRegularInstalments {
	var next LastRegularInstalments
	if withheld.NursingCare {
		_, next.NursingCare = RegularInstalmentOf(s.NursingCare, s.NursingCareLastRegular)
		next.NursingCareStarted = true
	}
	if withheld.LateElderly {
		_, next.LateElderly = RegularInstalmentOf(s.LateElderly, s.LateElderlyLastRegular)
		next.LateElderlyStarted = true
	}
	return next
}

func (s SpecialCollectionSubject) From(last LastRegularInstalments) SpecialCollectionSubject {
	s.NursingCareLastRegular = last.NursingCare
	if !last.NursingCareStarted {
		s.NursingCareLastRegular = ProvisionalInstalmentOf(s.NursingCare)
	}
	s.LateElderlyLastRegular = last.LateElderly
	if !last.LateElderlyStarted {
		s.LateElderlyLastRegular = ProvisionalInstalmentOf(s.LateElderly)
	}
	return s
}
