package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codeartifact"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	resource.AddTestSweepers("aws_codeartifact_repository", &resource.Sweeper{
		Name: "aws_codeartifact_repository",
		F:    testSweepCodeArtifactRepositories,
	})
}

func testSweepCodeArtifactRepositories(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).codeartifactconn
	input := &codeartifact.ListRepositoriesInput{}
	var sweeperErrs *multierror.Error

	err = conn.ListRepositoriesPages(input, func(page *codeartifact.ListRepositoriesOutput, lastPage bool) bool {
		for _, repositoryPtr := range page.Repositories {
			if repositoryPtr == nil {
				continue
			}

			repository := aws.StringValue(repositoryPtr.Name)
			input := &codeartifact.DeleteRepositoryInput{
				Repository:  repositoryPtr.Name,
				Domain:      repositoryPtr.DomainName,
				DomainOwner: repositoryPtr.DomainOwner,
			}

			log.Printf("[INFO] Deleting CodeArtifact Repository: %s", repository)

			_, err := conn.DeleteRepository(input)

			if err != nil {
				sweeperErr := fmt.Errorf("error deleting CodeArtifact Repository (%s): %w", repository, err)
				log.Printf("[ERROR] %s", sweeperErr)
				sweeperErrs = multierror.Append(sweeperErrs, sweeperErr)
			}
		}

		return !lastPage
	})

	if testSweepSkipSweepError(err) {
		log.Printf("[WARN] Skipping CodeArtifact Repository sweep for %s: %s", region, err)
		return nil
	}

	if err != nil {
		return fmt.Errorf("error listing CodeArtifact Repositories: %w", err)
	}

	return sweeperErrs.ErrorOrNil()
}

func TestAccAWSCodeArtifactRepository_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_codeartifact_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeArtifactRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeArtifactRepositoryBasicConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					testAccCheckResourceAttrRegionalARN(resourceName, "arn", "codeartifact", fmt.Sprintf("repository/%s/%s", rName, rName)),
					resource.TestCheckResourceAttr(resourceName, "repository", rName),
					resource.TestCheckResourceAttr(resourceName, "domain", rName),
					testAccCheckResourceAttrAccountID(resourceName, "domain_owner"),
					testAccCheckResourceAttrAccountID(resourceName, "administrator_account"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "upstreams.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSCodeArtifactRepository_description(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_codeartifact_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeArtifactRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeArtifactRepositoryDescConfig(rName, "desc"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "desc"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCodeArtifactRepositoryDescConfig(rName, "desc2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "description", "desc2"),
				),
			},
		},
	})
}

func TestAccAWSCodeArtifactRepository_upstreams(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_codeartifact_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeArtifactRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeArtifactRepositoryUpstreamsConfig1(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "upstreams.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "upstreams.0.repository_name", fmt.Sprintf("%s-upstream1", rName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSCodeArtifactRepositoryUpstreamsConfig2(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "upstreams.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "upstreams.0.repository_name", fmt.Sprintf("%s-upstream1", rName)),
					resource.TestCheckResourceAttr(resourceName, "upstreams.1.repository_name", fmt.Sprintf("%s-upstream2", rName)),
				),
			},
			{
				Config: testAccAWSCodeArtifactRepositoryUpstreamsConfig1(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "upstreams.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "upstreams.0.repository_name", fmt.Sprintf("%s-upstream1", rName)),
				),
			},
		},
	})
}

func TestAccAWSCodeArtifactRepository_disappears(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "aws_codeartifact_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodeArtifactRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCodeArtifactRepositoryBasicConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSCodeArtifactRepositoryExists(resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsCodeArtifactRepository(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSCodeArtifactRepositoryExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no CodeArtifact repository set")
		}

		conn := testAccProvider.Meta().(*AWSClient).codeartifactconn
		owner, domain, repo, err := decodeCodeArtifactRepositoryID(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, err = conn.DescribeRepository(&codeartifact.DescribeRepositoryInput{
			Repository:  aws.String(repo),
			Domain:      aws.String(domain),
			DomainOwner: aws.String(owner),
		})

		return err
	}
}

func testAccCheckAWSCodeArtifactRepositoryDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_codeartifact_repository" {
			continue
		}

		owner, domain, repo, err := decodeCodeArtifactRepositoryID(rs.Primary.ID)
		if err != nil {
			return err
		}
		conn := testAccProvider.Meta().(*AWSClient).codeartifactconn
		resp, err := conn.DescribeRepository(&codeartifact.DescribeRepositoryInput{
			Repository:  aws.String(repo),
			Domain:      aws.String(domain),
			DomainOwner: aws.String(owner),
		})

		if err == nil {
			if aws.StringValue(resp.Repository.Name) == repo &&
				aws.StringValue(resp.Repository.DomainName) == domain &&
				aws.StringValue(resp.Repository.DomainOwner) == owner {
				return fmt.Errorf("CodeArtifact Repository %s in Domain %s still exists", repo, domain)
			}
		}

		if isAWSErr(err, codeartifact.ErrCodeResourceNotFoundException, "") {
			return nil
		}

		return err
	}

	return nil
}

func testAccAWSCodeArtifactRepositoryBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description             = %[1]q
  deletion_window_in_days = 7
}

resource "aws_codeartifact_domain" "test" {
  domain         = %[1]q
  encryption_key = aws_kms_key.test.arn
}

resource "aws_codeartifact_repository" "test" {
  repository = %[1]q
  domain     = aws_codeartifact_domain.test.domain
}
`, rName)
}

func testAccAWSCodeArtifactRepositoryDescConfig(rName, desc string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description             = %[1]q
  deletion_window_in_days = 7
}

resource "aws_codeartifact_domain" "test" {
  domain         = %[1]q
  encryption_key = aws_kms_key.test.arn
}

resource "aws_codeartifact_repository" "test" {
  repository  = %[1]q
  domain      = aws_codeartifact_domain.test.domain
  description = %[2]q
}
`, rName, desc)
}

func testAccAWSCodeArtifactRepositoryUpstreamsConfig1(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description             = %[1]q
  deletion_window_in_days = 7
}

resource "aws_codeartifact_domain" "test" {
  domain         = %[1]q
  encryption_key = aws_kms_key.test.arn
}

resource "aws_codeartifact_repository" "upstream1" {
  repository = "%[1]s-upstream1"
  domain     = aws_codeartifact_domain.test.domain
}

resource "aws_codeartifact_repository" "test" {
  repository = %[1]q
  domain     = aws_codeartifact_domain.test.domain

  upstreams {
    repository_name = aws_codeartifact_repository.upstream1.repository
  }
}
`, rName)
}

func testAccAWSCodeArtifactRepositoryUpstreamsConfig2(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description             = %[1]q
  deletion_window_in_days = 7
}

resource "aws_codeartifact_domain" "test" {
  domain         = %[1]q
  encryption_key = aws_kms_key.test.arn
}

resource "aws_codeartifact_repository" "upstream1" {
  repository = "%[1]s-upstream1"
  domain     = aws_codeartifact_domain.test.domain
}

resource "aws_codeartifact_repository" "upstream2" {
  repository = "%[1]s-upstream2"
  domain     = aws_codeartifact_domain.test.domain
}

resource "aws_codeartifact_repository" "test" {
  repository = %[1]q
  domain     = aws_codeartifact_domain.test.domain

  upstreams {
    repository_name = aws_codeartifact_repository.upstream1.repository
  }

  upstreams {
    repository_name = aws_codeartifact_repository.upstream2.repository
  }
}
`, rName)
}
